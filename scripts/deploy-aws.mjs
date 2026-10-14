import { spawnSync } from "node:child_process";
import { existsSync, mkdirSync } from "node:fs";
import { fileURLToPath, pathToFileURL } from "node:url";

const repositoryRoot = fileURLToPath(new URL("../", import.meta.url));
process.chdir(repositoryRoot);

const dryRun = process.argv.includes("--dry-run");
const planOnly = process.argv.includes("--plan-only");
const unknownArguments = process.argv.slice(2).filter((argument) => !["--dry-run", "--plan-only"].includes(argument));
if (unknownArguments.length > 0) {
  throw new Error(`unknown deployment arguments: ${unknownArguments.join(", ")}`);
}

function readEnvironment(name, fallback = "") {
  const value = (process.env[name] ?? "").trim();
  return value === "" ? fallback : value;
}

const configuration = {
  awsProfile: readEnvironment("AWS_PROFILE"),
  awsRegion: readEnvironment("AWS_REGION", "ap-southeast-2"),
  stackName: readEnvironment("STACK_NAME", "npt-shortenlink-prod"),
  domainName: readEnvironment("DOMAIN_NAME", "npt-shortenlink.dev"),
  certificateArn: readEnvironment("CERTIFICATE_ARN"),
};

function assertMatches(name, value, expression, hint) {
  if (!expression.test(value)) {
    throw new Error(`${name} is invalid. ${hint}`);
  }
}

function validateConfiguration() {
  const missing = ["certificateArn"].filter(
    (key) => configuration[key] === "",
  );
  if (missing.length > 0) {
    const environmentNames = ["CERTIFICATE_ARN"];
    throw new Error(
      `missing required deployment values: ${environmentNames.join(", ")}`,
    );
  }

  assertMatches(
    "AWS_REGION",
    configuration.awsRegion,
    /^[a-z]{2}(?:-gov)?-[a-z]+-\d$/,
    "Use the project's selected region, such as ap-southeast-2.",
  );
  if (configuration.awsProfile !== "") {
    assertMatches(
      "AWS_PROFILE",
      configuration.awsProfile,
      /^[A-Za-z0-9_-]+$/,
      "Use letters, numbers, underscores, or hyphens.",
    );
  }
  assertMatches(
    "STACK_NAME",
    configuration.stackName,
    /^[a-z][a-z0-9-]{1,39}$/,
    "Use 2-40 lowercase letters, numbers, or hyphens.",
  );
  assertMatches(
    "DOMAIN_NAME",
    configuration.domainName,
    /^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/,
    "Use a lower-case DNS name.",
  );
  assertMatches(
    "CERTIFICATE_ARN",
    configuration.certificateArn,
    /^arn:aws:acm:us-east-1:\d{12}:certificate\/[0-9a-fA-F-]+$/,
    "CloudFront requires an ACM certificate ARN from us-east-1.",
  );


}

const windowsCommandCache = new Map();

function platformCommand(command) {
  if (process.platform !== "win32") {
    return command;
  }
  if (windowsCommandCache.has(command)) {
    return windowsCommandCache.get(command);
  }
  const commandName =
    {
      aws: "aws.exe",
      go: "go.exe",
      pnpm: "pnpm.cmd",
      terraform: "terraform.exe",
    }[command] ?? command;
  const lookup = spawnSync("where.exe", [commandName], {
    encoding: "utf8",
    stdio: ["ignore", "pipe", "ignore"],
  });
  const resolved =
    lookup.status === 0 ? lookup.stdout.trim().split(/\r?\n/u)[0] : commandName;
  windowsCommandCache.set(command, resolved);
  return resolved;
}

function displayCommand(command, args) {
  return [command, ...args]
    .map((value) => (/[\s"]/u.test(value) ? JSON.stringify(value) : value))
    .join(" ");
}

function windowsShellCommand(command, args) {
  return [platformCommand(command), ...args]
    .map((value) => `"${value.replaceAll('"', '""')}"`)
    .join(" ");
}

function execute(command, args, options = {}) {
  const { capture = false, executeDuringDryRun = false, env = {} } = options;
  console.log(`\n> ${displayCommand(command, args)}`);

  if (dryRun && !executeDuringDryRun) {
    return "";
  }

  const useWindowsShell = process.platform === "win32";
  const result = spawnSync(
    useWindowsShell
      ? windowsShellCommand(command, args)
      : platformCommand(command),
    useWindowsShell ? [] : args,
    {
    cwd: repositoryRoot,
    encoding: capture ? "utf8" : undefined,
    env: { ...process.env, ...env },
    shell: useWindowsShell,
    stdio: capture ? ["ignore", "pipe", "inherit"] : "inherit",
    },
  );
  if (result.error) {
    throw new Error(`could not run ${command}: ${result.error.message}`);
  }
  if (result.status !== 0) {
    throw new Error(`${command} exited with status ${result.status}`);
  }
  return capture ? result.stdout.trim() : "";
}

function awsArguments(...arguments_) {
  return configuration.awsProfile
    ? ["--profile", configuration.awsProfile, ...arguments_]
    : arguments_;
}

function verifyToolchain() {
  execute("go", ["version"], { executeDuringDryRun: true });
  execute("pnpm", ["--version"], { executeDuringDryRun: true });
  execute("terraform", ["version"], { executeDuringDryRun: true });
  execute("aws", ["--version"], { executeDuringDryRun: true });
}

function terraformOutput(name) {
  if (dryRun) return `DRY_RUN_${name}`;
  return execute("terraform", ["-chdir=infra/aws", "output", "-raw", name], { capture: true });
}

export function assertDeploymentPermissions(evaluations) {
  if (!Array.isArray(evaluations) || evaluations.length === 0) {
    throw new Error("AWS permission evaluation is empty");
  }
  if (evaluations.some((entry) => !entry.EvalActionName || !["allowed", "implicitDeny", "explicitDeny"].includes(entry.EvalDecision))) {
    throw new Error("AWS permission evaluation is malformed");
  }
  const denied = evaluations.filter((entry) => entry.EvalDecision === "explicitDeny" || entry.OrganizationsDecisionDetail?.AllowedByOrganizations === false);
  if (denied.length > 0) {
    throw new Error(`Deployment denied for ${denied.map((entry) => entry.EvalActionName).join(", ")}; check AWS Organizations SCP and IAM policies before applying`);
  }
}

function verifyDeploymentPermissions(identity) {
  // IAM simulation cannot evaluate assumed-role sessions; AWS still enforces permissions during apply.
  if (!identity.Arn.includes(":user/")) return;
  const checks = [
    [configuration.awsRegion, ["lambda:CreateFunction", "dynamodb:CreateTable", "apigateway:POST", "s3:CreateBucket"]],
    ["us-east-1", ["cloudfront:CreateDistribution", "iam:CreateRole", "iam:PassRole"]],
  ];
  for (const [region, actions] of checks) {
    const result = execute("aws", awsArguments(
      "iam", "simulate-principal-policy", "--policy-source-arn", identity.Arn,
      "--action-names", ...actions,
      "--context-entries", `ContextKeyName=aws:RequestedRegion,ContextKeyValues=${region},ContextKeyType=string`, "--output", "json",
    ), { capture: true });
    assertDeploymentPermissions(JSON.parse(result).EvaluationResults);
  }
}

export async function verifyLinkFlow(baseUrl) {
  const target = "https://example.com/";
  const created = await fetch(`${baseUrl}/api/v1/links`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url: target, expires_in_days: 1 }),
    signal: AbortSignal.timeout(10_000),
  });
  if (created.status !== 201) throw new Error("Link creation smoke test failed");
  const code = (await created.json()).data?.code;
  if (typeof code !== "string" || !/^[a-z0-9-]{4,32}$/.test(code)) {
    throw new Error("Link creation returned an invalid code");
  }
  const metadata = await fetch(`${baseUrl}/api/v1/links/${code}`, { signal: AbortSignal.timeout(10_000) });
  const data = (await metadata.json()).data;
  if (metadata.status !== 200 || data?.code !== code || data?.status !== "active") {
    throw new Error("Link metadata smoke test failed");
  }
  const redirect = await fetch(`${baseUrl}/link/${code}`, { redirect: "manual", signal: AbortSignal.timeout(10_000) });
  if (redirect.status !== 302 || redirect.headers.get("Location") !== target) {
    throw new Error("Link redirect smoke test failed");
  }
  console.log("Smoke test passed: create -> metadata -> redirect");
}

async function smokeTest(url, label, validateResponse) {
  const attempts = 18;
  for (let attempt = 1; attempt <= attempts; attempt += 1) {
    try {
      const response = await fetch(url, {
        redirect: "manual",
        signal: AbortSignal.timeout(10_000),
      });
      if (await validateResponse(response)) {
        console.log(`Smoke test passed: ${label} (${url})`);
        return;
      }
      console.warn(
        `Smoke attempt ${attempt}/${attempts} returned HTTP ${response.status}: ${label}`,
      );
    } catch (error) {
      console.warn(
        `Smoke attempt ${attempt}/${attempts} failed: ${label}: ${error.message}`,
      );
    }
    if (attempt < attempts) {
      await new Promise((resolve) => setTimeout(resolve, 10_000));
    }
  }
  throw new Error(`smoke test did not pass: ${label}`);
}

async function main() {
  validateConfiguration();
  console.log(
    `${dryRun ? "Dry-running" : "Deploying"} ${configuration.stackName} in ${configuration.awsRegion}`,
  );

  verifyToolchain();

  let identity;
  if (!dryRun) {
    identity = JSON.parse(execute("aws", awsArguments("sts", "get-caller-identity", "--output", "json"), { capture: true }));
    const certificate = JSON.parse(execute("aws", awsArguments("acm", "describe-certificate", "--certificate-arn", configuration.certificateArn, "--region", "us-east-1", "--output", "json"), { capture: true })).Certificate;
    if (certificate.Status !== "ISSUED" || !certificate.SubjectAlternativeNames.includes(configuration.domainName)) {
      throw new Error("The ACM certificate must be issued and cover DOMAIN_NAME");
    }
    console.log(`Authenticated to AWS account ${identity.Account}`);
  }

  execute("pnpm", ["install", "--frozen-lockfile", "--ignore-scripts"]);
  execute("pnpm", ["verify"]);
  execute("pnpm", ["audit:deps"]);
  execute("pnpm", ["vuln:api"]);
  execute("pnpm", ["--package=@redocly/cli@2.40.0", "dlx", "redocly", "lint", "openapi/openapi.yaml"]);
  if (!dryRun) mkdirSync("infra/aws/.build", { recursive: true });
  execute("go", ["build", "-trimpath", "-ldflags=-s -w", "-o", "infra/aws/.build/bootstrap", "./services/shortener-api/cmd/api"], { env: { GOOS: "linux", GOARCH: "arm64", CGO_ENABLED: "0" } });
  process.env.TF_VAR_aws_region = configuration.awsRegion;
  process.env.TF_VAR_deployment_name = configuration.stackName;
  process.env.TF_VAR_domain_name = configuration.domainName;
  process.env.TF_VAR_certificate_arn = configuration.certificateArn;
  execute("terraform", ["-chdir=infra/aws", "init", "-input=false"]);
  execute("terraform", ["-chdir=infra/aws", "fmt", "-check", "-recursive"]);
  execute("terraform", ["-chdir=infra/aws", "validate"]);
  execute("terraform", ["-chdir=infra/aws", "test"]);
  execute("terraform", ["-chdir=infra/aws", "plan", "-input=false", "-out=production.tfplan"]);
  if (planOnly) {
    console.log("Terraform plan saved; no resources were applied or frontend files uploaded.");
    return;
  }
  if (!dryRun) verifyDeploymentPermissions(identity);
  execute("terraform", ["-chdir=infra/aws", "apply", "-input=false", "production.tfplan"]);

  const frontendBucket = terraformOutput("frontend_bucket_name");
  const distributionId = terraformOutput("cloudfront_distribution_id");
  const siteUrl = `https://${configuration.domainName}`;
  const cloudfrontDomain = terraformOutput("cloudfront_domain_name");
  console.log(`Name.com DNS: ANAME, Host empty (apex), Answer ${cloudfrontDomain}, TTL 300`);

  if (!dryRun && !existsSync("apps/web/out/_next/static")) {
    throw new Error("Next.js static assets are missing from apps/web/out");
  }

  execute("aws", [
    ...awsArguments(
      "s3",
      "sync",
      "apps/web/out/_next/static",
      `s3://${frontendBucket}/_next/static`,
      "--cache-control",
      "public,max-age=31536000,immutable",
      "--region",
      configuration.awsRegion,
    ),
  ]);
  execute("aws", [
    ...awsArguments(
      "s3",
      "sync",
      "apps/web/out",
      `s3://${frontendBucket}`,
      "--delete",
      "--exclude",
      "_next/static/*",
      "--cache-control",
      "public,max-age=0,must-revalidate",
      "--region",
      configuration.awsRegion,
    ),
  ]);
  const invalidationId = dryRun
    ? "DRY_RUN_INVALIDATION"
    : execute(
        "aws",
        awsArguments(
          "cloudfront",
          "create-invalidation",
          "--distribution-id",
          distributionId,
          "--paths",
          "/*",
          "--query",
          "Invalidation.Id",
          "--output",
          "text",
        ),
        { capture: true },
      );
  if (dryRun) {
    execute("aws", [
      ...awsArguments(
        "cloudfront",
        "create-invalidation",
        "--distribution-id",
        distributionId,
        "--paths",
        "/*",
        "--query",
        "Invalidation.Id",
        "--output",
        "text",
      ),
    ]);
  }
  execute("aws", [
    ...awsArguments(
      "cloudfront",
      "wait",
      "invalidation-completed",
      "--distribution-id",
      distributionId,
      "--id",
      invalidationId,
    ),
  ]);

  if (!dryRun) {
    const verificationUrl = `https://${cloudfrontDomain}`;
    await smokeTest(verificationUrl, "frontend", (response) =>
      Promise.resolve(response.status === 200),
    );
    await smokeTest(`${verificationUrl}/healthz`, "API health", async (response) => {
      if (response.status !== 200) {
        return false;
      }
      const payload = await response.json().catch(() => null);
      return payload?.status === "ok";
    });
    await verifyLinkFlow(verificationUrl);
  }

  console.log(
    `\n${dryRun ? "Deployment plan is valid" : `AWS deployment completed; set Name.com DNS to enable ${siteUrl}`}`,
  );
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  await main();
}
