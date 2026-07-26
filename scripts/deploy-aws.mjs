import { spawnSync } from "node:child_process";
import { existsSync } from "node:fs";
import { fileURLToPath } from "node:url";

const repositoryRoot = fileURLToPath(new URL("../", import.meta.url));
process.chdir(repositoryRoot);

const dryRun = process.argv.includes("--dry-run");
const unknownArguments = process.argv.slice(2).filter((argument) => argument !== "--dry-run");
if (unknownArguments.length > 0) {
  throw new Error(`unknown deployment arguments: ${unknownArguments.join(", ")}`);
}

function readEnvironment(name, fallback = "") {
  const value = (process.env[name] ?? "").trim();
  return value === "" ? fallback : value;
}

const configuration = {
  awsProfile: readEnvironment("AWS_PROFILE"),
  awsRegion: readEnvironment("AWS_REGION", "ap-southeast-1"),
  stackName: readEnvironment("STACK_NAME", "npt-shortenlink-prod"),
  environmentName: readEnvironment("ENVIRONMENT_NAME", "prod"),
  domainName: readEnvironment("DOMAIN_NAME", "npt-shortenlink.dev"),
  certificateArn: readEnvironment("CERTIFICATE_ARN"),
  hostedZoneId: readEnvironment("HOSTED_ZONE_ID"),
};
configuration.corsAllowedOrigins = readEnvironment(
  "CORS_ALLOWED_ORIGINS",
  `https://${configuration.domainName}`,
);

function assertMatches(name, value, expression, hint) {
  if (!expression.test(value)) {
    throw new Error(`${name} is invalid. ${hint}`);
  }
}

function validateConfiguration() {
  const missing = ["certificateArn", "hostedZoneId"].filter(
    (key) => configuration[key] === "",
  );
  if (missing.length > 0) {
    const environmentNames = missing.map((key) =>
      key === "certificateArn" ? "CERTIFICATE_ARN" : "HOSTED_ZONE_ID",
    );
    throw new Error(
      `missing required deployment values: ${environmentNames.join(", ")}`,
    );
  }

  assertMatches(
    "AWS_REGION",
    configuration.awsRegion,
    /^[a-z]{2}(?:-gov)?-[a-z]+-\d$/,
    "Use an AWS region such as ap-southeast-1.",
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
    /^[A-Za-z][A-Za-z0-9-]{0,127}$/,
    "CloudFormation stack names must start with a letter.",
  );
  assertMatches(
    "ENVIRONMENT_NAME",
    configuration.environmentName,
    /^[a-z][a-z0-9-]{1,15}$/,
    "Use 2-16 lower-case letters, numbers, or hyphens.",
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
  assertMatches(
    "HOSTED_ZONE_ID",
    configuration.hostedZoneId,
    /^Z[A-Z0-9]+$/,
    "Use the Route 53 public hosted-zone ID.",
  );

  for (const origin of configuration.corsAllowedOrigins.split(",")) {
    let parsed;
    try {
      parsed = new URL(origin.trim());
    } catch {
      throw new Error(`CORS_ALLOWED_ORIGINS contains an invalid URL: ${origin}`);
    }
    if (
      !["http:", "https:"].includes(parsed.protocol) ||
      parsed.username !== "" ||
      parsed.password !== "" ||
      parsed.pathname !== "/" ||
      parsed.search !== "" ||
      parsed.hash !== ""
    ) {
      throw new Error(
        `CORS_ALLOWED_ORIGINS must contain origins only: ${origin}`,
      );
    }
  }
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
      sam: "sam.cmd",
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
  const { capture = false, executeDuringDryRun = false } = options;
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
    env: process.env,
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

function samProfileArguments() {
  return configuration.awsProfile
    ? ["--profile", configuration.awsProfile]
    : [];
}

function verifyToolchain() {
  execute("go", ["version"], { executeDuringDryRun: true });
  execute("pnpm", ["--version"], { executeDuringDryRun: true });
  execute("sam", ["--version"], { executeDuringDryRun: true });
  execute("aws", ["--version"], { executeDuringDryRun: true });
}

function stackOutput(outputKey) {
  if (dryRun) {
    return `DRY_RUN_${outputKey}`;
  }
  const output = execute(
    "aws",
    awsArguments(
      "cloudformation",
      "describe-stacks",
      "--stack-name",
      configuration.stackName,
      "--region",
      configuration.awsRegion,
      "--query",
      `Stacks[0].Outputs[?OutputKey==\`${outputKey}\`].OutputValue | [0]`,
      "--output",
      "text",
    ),
    { capture: true },
  );
  if (output === "" || output === "None") {
    throw new Error(`CloudFormation output ${outputKey} is missing`);
  }
  return output;
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

  if (!dryRun) {
    const account = execute(
      "aws",
      awsArguments(
        "sts",
        "get-caller-identity",
        "--query",
        "Account",
        "--output",
        "text",
      ),
      { capture: true },
    );
    console.log(`Authenticated to AWS account ${account}`);
  }

  execute("pnpm", ["install", "--frozen-lockfile"]);
  execute("pnpm", ["verify"]);
  execute("pnpm", ["audit:deps"]);
  execute("pnpm", ["vuln:api"]);
  execute("pnpm", [
    "--package=@redocly/cli@2.40.0",
    "dlx",
    "redocly",
    "lint",
    "openapi/openapi.yaml",
  ]);
  execute("sam", [
    "validate",
    "--lint",
    "--template-file",
    "infra/aws/template.yaml",
  ]);
  execute("sam", [
    "build",
    "--template-file",
    "infra/aws/template.yaml",
  ]);

  execute("sam", [
    "deploy",
    "--template-file",
    ".aws-sam/build/template.yaml",
    "--stack-name",
    configuration.stackName,
    "--region",
    configuration.awsRegion,
    "--resolve-s3",
    "--capabilities",
    "CAPABILITY_IAM",
    "--no-confirm-changeset",
    "--no-fail-on-empty-changeset",
    "--parameter-overrides",
    `EnvironmentName=${configuration.environmentName}`,
    `DomainName=${configuration.domainName}`,
    `CertificateArn=${configuration.certificateArn}`,
    `HostedZoneId=${configuration.hostedZoneId}`,
    `CorsAllowedOrigins=${configuration.corsAllowedOrigins}`,
    ...samProfileArguments(),
  ]);

  const frontendBucket = stackOutput("FrontendBucketName");
  const distributionId = stackOutput("CloudFrontDistributionId");
  const siteUrl = dryRun
    ? `https://${configuration.domainName}`
    : stackOutput("SiteUrl");

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
    await smokeTest(siteUrl, "frontend", (response) =>
      Promise.resolve(response.status === 200),
    );
    await smokeTest(`${siteUrl}/healthz`, "API health", async (response) => {
      if (response.status !== 200) {
        return false;
      }
      const payload = await response.json().catch(() => null);
      return payload?.status === "ok";
    });
  }

  console.log(
    `\n${dryRun ? "Deployment plan is valid" : `Deployment completed: ${siteUrl}`}`,
  );
}

await main();
