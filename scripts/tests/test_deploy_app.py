"""Run production orchestration with Terraform/AWS/build process boundaries."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
FIXTURE = r'''#!/usr/bin/env python3
import hashlib,json,os,sys
from pathlib import Path
root=Path(os.environ['FIXTURE_ROOT']);cmd=Path(sys.argv[0]).name;args=sys.argv[1:];case=os.environ['CASE']
with (root/'calls').open('a') as f:f.write(json.dumps([cmd,*args])+'\n')
def outputs():
 return {k:{'value':v} for k,v in {
 'resolve_function_name':'url-shortener-dev-resolve-link','resolve_published_version':'2',
 'codedeploy_application_name':'url-shortener-dev-resolve','codedeploy_deployment_group_name':'url-shortener-dev-resolve-live',
 'frontend_bucket_name':'url-shortener-dev-111111111111-frontend','cloudfront_distribution_id':'EDIST',
 'frontend_url':'https://dtest.cloudfront.net','http_api_endpoint':('https://api2.execute-api.ap-southeast-1.amazonaws.com' if case=='config-changed' and (root/'deployed').exists() else 'https://api.execute-api.ap-southeast-1.amazonaws.com'),
 'websocket_endpoint':'wss://ws.execute-api.ap-southeast-1.amazonaws.com/prod',
 'cognito_login_domain':'https://test.auth.ap-southeast-1.amazoncognito.com','cognito_client_id':'testclient'}.items()}
if cmd=='terraform':
 if 'output' in args:
  if '-raw' in args:print(outputs()[args[-1]]['value'])
  else:print(json.dumps(outputs()))
 elif 'plan' in args:
  Path(next(x.removeprefix('-out=') for x in args if x.startswith('-out='))).write_bytes(b'secret plan')
 elif 'show' in args:print(json.dumps({'resource_changes':[{'address':'module.foo','change':{'actions':['update'],'after':{'secret':'NEVER PRINT'}}}]}))
 elif 'apply' in args:
  if case=='apply-failed':sys.exit(4)
  (root/'applied').touch()
elif cmd=='corepack':
 if 'build' in args:
  out=root/'frontend/out'
  for page in ['index.html','login/index.html','auth/callback/index.html','dashboard/index.html','architecture/index.html','_next/static/test.js']:
   p=out/page;p.parent.mkdir(parents=True,exist_ok=True);p.write_text('static')
  if case=='build-failed':sys.exit(5)
elif cmd=='node':pass
elif cmd=='aws':
 if args[:2]==['sts','get-caller-identity']:print('111111111111')
 elif args[:2]==['deploy','list-deployments']:print('{"deployments":[]}')
 elif args[:2]==['lambda','get-alias']:print(json.dumps({'FunctionVersion':'2' if (root/'deployed').exists() and case!='rollback' else '1'}))
 elif args[:2]==['deploy','create-deployment']:(root/'deployed').touch();print('{"deploymentId":"d-new"}')
 elif args[:2]==['deploy','get-deployment']:
  rollback=args[args.index('--deployment-id')+1]=='d-rollback'
  print(json.dumps({'deploymentInfo':{'status':'Stopped' if case=='rollback' and not rollback else 'Succeeded','rollbackInfo':{'rollbackDeploymentId':'d-rollback'} if case=='rollback' and not rollback else {}}}))
 elif args[:2]==['s3','cp']:
  dest=Path(args[3]);dest.parent.mkdir(parents=True,exist_ok=True)
  if args[2].endswith('manifest.json'):
   packages=''.join(hashlib.sha256(p.read_bytes()).hexdigest()+'  '+str(p.relative_to(root))+'\n' for p in sorted((root/'backend/dist').glob('*.zip')))
   dest.write_text(json.dumps({'commit':'b'*40 if case=='manifest-mismatch' else 'a'*40,'repository':'TrieuNguyenPhu/shorten-link','packages':packages,'plan_sha256':hashlib.sha256(b'plan').hexdigest()}))
  elif args[2].endswith('app.tfplan'):dest.write_bytes(b'corrupt' if case=='digest-mismatch' else b'plan')
 elif args[:2]==['cloudfront','create-invalidation']:print('I123')
 elif args[:2] in [['s3','sync'],['s3','rm'],['s3api','put-object'],['cloudfront','wait']]:pass
 else:sys.exit('Unexpected AWS call '+str(args))
else:sys.exit('Unexpected process '+cmd)
'''

class AppReleaseTests(unittest.TestCase):
    def run_case(self, case='success', mode='apply'):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);(root/'scripts').mkdir();(root/'bin').mkdir();(root/'backend/dist').mkdir(parents=True)
            (root/'backend/dist/resolve-link.zip').write_bytes(b'zip')
            for name in ['deploy-app.sh','deploy-canary.sh','publish-frontend.sh']:
                shutil.copy2(ROOT/'scripts'/name,root/'scripts'/name)
            for name in ['terraform','aws','corepack','node']:
                p=root/'bin'/name;p.write_text(FIXTURE);p.chmod(0o755)
            env=dict(os.environ,PATH=f"{root/'bin'}:{os.environ['PATH']}",FIXTURE_ROOT=tmp,CASE=case,
                     TF_STATE_BUCKET='test-state-bucket',AWS_ACCOUNT_ID='111111111111',GITHUB_SHA='a'*40,
                     GITHUB_REPOSITORY='TrieuNguyenPhu/shorten-link',GITHUB_REF='refs/heads/main',
                     CANARY_TIMEOUT_SECONDS='2',CANARY_POLL_SECONDS='1')
            result=subprocess.run(['bash',str(root/'scripts/deploy-app.sh'),mode,'123'],env=env,capture_output=True,text=True,timeout=15)
            calls=[json.loads(s) for s in (root/'calls').read_text().splitlines()]
            return result,calls
    def test_plan_never_applies_or_publishes_and_omits_secret_values(self):
        result,calls=self.run_case(mode='plan')
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertNotIn('NEVER PRINT',result.stdout)
        self.assertFalse(any('apply' in c or 'create-deployment' in c or 'sync' in c for c in calls))
        self.assertEqual(sum('--if-none-match' in c for c in calls),2)
    def test_success_builds_then_canary_then_assets_data_html_and_invalidation(self):
        result,calls=self.run_case()
        self.assertEqual(result.returncode,0,result.stderr)
        build=next(i for i,c in enumerate(calls) if c[0]=='corepack' and 'build' in c)
        canary=next(i for i,c in enumerate(calls) if 'create-deployment' in c)
        upload=next(i for i,c in enumerate(calls) if c[:3]==['aws','s3','sync'])
        invalidation=next(i for i,c in enumerate(calls) if 'create-invalidation' in c)
        self.assertLess(build,canary);self.assertLess(canary,upload);self.assertLess(upload,invalidation)
        uploads=[c for c in calls if c[:3]==['aws','s3','cp'] and '--recursive' in c]
        self.assertIn('_next/static/*',uploads[0]);self.assertIn('--include',uploads[1])
    def test_wrong_or_corrupt_plan_never_applies(self):
        for case in ['manifest-mismatch','digest-mismatch']:
            result,calls=self.run_case(case)
            self.assertNotEqual(result.returncode,0,case)
            self.assertFalse(any('apply' in c or 'create-deployment' in c or 'sync' in c for c in calls))
    def test_failure_and_rollback_block_frontend_upload(self):
        for case in ['apply-failed','build-failed','rollback','config-changed']:
            result,calls=self.run_case(case)
            self.assertNotEqual(result.returncode,0,case)
            self.assertFalse(any('sync' in c or 'create-invalidation' in c for c in calls))
            if case not in ['rollback','config-changed']:self.assertFalse(any('create-deployment' in c for c in calls))

if __name__=='__main__':unittest.main()
