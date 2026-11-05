"""Exercise the real Bash writer with a CLI boundary fixture; never call AWS."""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]

class CanaryTests(unittest.TestCase):
    def run_case(self, case, target='2', extra=()):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            outputs = {k: {'value': v} for k, v in {
                'resolve_function_name': 'url-shortener-dev-resolve-link',
                'resolve_published_version': target,
                'codedeploy_application_name': 'url-shortener-dev-resolve',
                'codedeploy_deployment_group_name': 'url-shortener-dev-resolve-live',
            }.items()}
            (root / 'outputs.json').write_text(json.dumps(outputs))
            (root / 'aws').write_text('''#!/usr/bin/env python3
import json,os,sys
from pathlib import Path
args=sys.argv[1:];case=os.environ['CASE'];root=Path(os.environ['FIXTURE_ROOT'])
with (root/'calls').open('a') as f:f.write(json.dumps(args)+'\\n')
if args[:2]==['deploy','list-deployments']:print(json.dumps({'deployments':['d-active'] if case=='active' else []}))
elif args[:2]==['lambda','get-alias']:
 done=(root/'created').exists()
 print(json.dumps({'FunctionVersion':'2' if done and case=='success' else '1','RoutingConfig':{'AdditionalVersionWeights':{'2':0.1}} if case=='weighted' else {}}))
elif args[:2]==['deploy','create-deployment']:
 assert os.environ['AWS_MAX_ATTEMPTS']=='1'
 (root/'created').touch()
 if case=='lost':sys.exit(9)
 request=json.load(open(args[args.index('--cli-input-json')+1].removeprefix('file://')))
 spec=json.loads(request['revision']['appSpecContent']['content'])
 props=spec['Resources'][0]['Resolve']['Properties']
 assert props['CurrentVersion']=='1' and props['TargetVersion']=='2' and props['Alias']=='live'
 print('{"deploymentId":"d-new"}')
elif args[:2]==['deploy','get-deployment']:
 rollback=args[args.index('--deployment-id')+1]=='d-rollback'
 if case=='unknown':status='FutureState'
 elif case=='pending':status='InProgress'
 elif case in ['rollback','rollback-failed']:
  status=('Failed' if case=='rollback-failed' else 'Succeeded') if rollback else 'Stopped'
 else:status='Succeeded'
 print(json.dumps({'deploymentInfo':{'status':status,'rollbackInfo':{} if rollback else ({'rollbackDeploymentId':'d-rollback'} if case.startswith('rollback') else {})}}))
else:sys.exit('Unexpected CLI request: '+str(args))
''')
            (root / 'aws').chmod(0o755)
            env = dict(os.environ, PATH=f"{tmp}:{os.environ['PATH']}", CASE=case,
                       FIXTURE_ROOT=tmp, CANARY_POLL_SECONDS='1', CANARY_TIMEOUT_SECONDS='2')
            result = subprocess.run(['bash', str(ROOT/'scripts/deploy-canary.sh'), str(root/'outputs.json'), *extra],
                                    env=env, text=True, capture_output=True, timeout=10)
            calls = [json.loads(s) for s in (root/'calls').read_text().splitlines()] if (root/'calls').exists() else []
            return result, calls

    def test_same_version_does_not_create_deployment(self):
        result,calls=self.run_case('same',target='1')
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertFalse(any('create-deployment' in c for c in calls))
    def test_success_uses_current_alias_and_target(self):
        result,calls=self.run_case('success')
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertEqual(sum('create-deployment' in c for c in calls),1)
    def test_active_and_weighted_alias_fail_before_write(self):
        for case in ['active','weighted']:
            with self.subTest(case=case):
                result,calls=self.run_case(case)
                self.assertNotEqual(result.returncode,0)
                self.assertFalse(any('create-deployment' in c for c in calls))
    def test_check_only_never_writes(self):
        result,calls=self.run_case('success',extra=('--check-only',))
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertFalse(any('create-deployment' in c for c in calls))
    def test_rollback_is_waited_and_fails_release(self):
        for case in ['rollback','rollback-failed']:
            result,calls=self.run_case(case)
            self.assertNotEqual(result.returncode,0)
            self.assertTrue(any('d-rollback' in c for c in calls))
            self.assertFalse(any('update-alias' in c for c in calls))
    def test_uncertain_failed_or_timed_out_never_reset_alias(self):
        for case in ['lost','unknown','pending','bad-success']:
            result,calls=self.run_case(case)
            self.assertNotEqual(result.returncode,0,case)
            self.assertFalse(any('update-alias' in c for c in calls))
    def test_invalid_target_rejected_before_network(self):
        result,calls=self.run_case('success',target='$LATEST')
        self.assertNotEqual(result.returncode,0)
        self.assertEqual(calls,[])

if __name__ == '__main__': unittest.main()
