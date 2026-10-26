import { spawnSync } from "node:child_process";
import { join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const repository = fileURLToPath(new URL("../", import.meta.url));

function command(program, args, { capture = false, ...options } = {}) {
  const result = spawnSync(program, args, {
    cwd: repository,
    env: { ...process.env, AWS_PAGER: "" },
    stdio: capture ? ["ignore", "pipe", "inherit"] : "inherit",
    encoding: "utf8",
    ...options,
  });
  if (result.error || result.status !== 0) {
    throw new Error(
      result.error?.message || `${program} failed (${result.status})`,
    );
  }
  return result.stdout;
}

export function deployWeb(execute = command) {
  const outputs = JSON.parse(
    execute("terraform", ["-chdir=infra/terraform/dev", "output", "-json"], {
      capture: true,
    }),
  );
  const values = Object.fromEntries(
    Object.entries(outputs).map(([key, output]) => [key, output.value]),
  );
  const {
    site_url: site,
    api_url: api,
    web_bucket: bucket,
    distribution_id: id,
    aws_account_id: account,
    aws_region: region,
    cognito_client_id: cognitoClientId,
    cognito_domain_url: cognitoDomainUrl,
    cognito_issuer: cognitoIssuer,
  } = values;
  if (
    [
      site,
      api,
      bucket,
      id,
      account,
      region,
      cognitoClientId,
      cognitoDomainUrl,
      cognitoIssuer,
    ].some((value) => typeof value !== "string" || !value)
  ) {
    throw new Error(
      "Hosting outputs are incomplete. Finish/import the dev stack and apply its reviewed configuration first.",
    );
  }
  for (const origin of [site, api, cognitoDomainUrl]) {
    const url = new URL(origin);
    if (
      url.protocol !== "https:" ||
      url.origin !== origin ||
      url.username ||
      url.password
    )
      throw new Error(
        "Terraform site/API/Cognito output must be an HTTPS origin without a path.",
      );
  }
  const issuerUrl = new URL(cognitoIssuer);
  if (
    issuerUrl.protocol !== "https:" ||
    issuerUrl.username ||
    issuerUrl.password
  ) {
    throw new Error("Terraform Cognito issuer must be an HTTPS URL.");
  }
  if (
    !/^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$/.test(bucket) ||
    !/^[A-Z0-9]+$/.test(id) ||
    !/^\d{12}$/.test(account) ||
    !/^[a-z]{2}(-[a-z]+)+-\d+$/.test(region)
  ) {
    throw new Error("Invalid Terraform deployment target.");
  }
  const identity = JSON.parse(
    execute(
      "aws",
      ["sts", "get-caller-identity", "--region", region, "--output", "json"],
      { capture: true },
    ),
  );
  if (identity.Account !== account)
    throw new Error(
      "AWS CLI account differs from the Terraform deployment account.",
    );
  execute("aws", [
    "s3api",
    "head-bucket",
    "--bucket",
    bucket,
    "--expected-bucket-owner",
    account,
    "--region",
    region,
  ]);
  const { Distribution: distribution } = JSON.parse(
    execute(
      "aws",
      [
        "cloudfront",
        "get-distribution",
        "--id",
        id,
        "--region",
        "us-east-1",
        "--output",
        "json",
      ],
      { capture: true },
    ),
  );
  const config = distribution?.DistributionConfig;
  const origins =
    config?.Origins?.Items?.map((origin) => origin.DomainName) || [];
  const siteHost = new URL(site).hostname;
  if (
    distribution?.ARN !== `arn:aws:cloudfront::${account}:distribution/${id}` ||
    distribution.Status !== "Deployed" ||
    !config?.Enabled ||
    !origins.includes(`${bucket}.s3.${region}.amazonaws.com`) ||
    !origins.includes(new URL(api).hostname) ||
    !(
      distribution.DomainName === siteHost ||
      config.Aliases?.Items?.includes(siteHost)
    )
  ) {
    throw new Error(
      "CloudFront is not deployed with the Terraform account, site and S3/API origins.",
    );
  }
  const pnpmCLI = process.env.npm_execpath;
  if (!pnpmCLI) throw new Error("Run this command with pnpm deploy:web.");
  execute(process.execPath, [pnpmCLI, "build"], {
    env: {
      ...process.env,
      NEXT_PUBLIC_API_BASE_URL: site,
      NEXT_PUBLIC_COGNITO_CLIENT_ID: cognitoClientId,
      NEXT_PUBLIC_COGNITO_DOMAIN: cognitoDomainUrl,
      NEXT_PUBLIC_COGNITO_ISSUER: cognitoIssuer,
    },
  });
  const output = join(repository, "apps/web/out");
  const target = `s3://${bucket}/`;
  const sync = (source, destination, filters) =>
    execute("aws", [
      "s3",
      "sync",
      source,
      destination,
      "--region",
      region,
      "--no-follow-symlinks",
      ...filters,
    ]);
  // Retain old fingerprinted assets so existing pages and rollback still work.
  sync(join(output, "_next/static"), `${target}_next/static/`, [
    "--cache-control",
    "public,max-age=31536000,immutable",
  ]);
  sync(output, target, [
    "--exclude",
    "_next/static/*",
    "--exclude",
    "*.html",
    "--cache-control",
    "no-cache",
  ]);
  sync(output, target, [
    "--exclude",
    "*",
    "--include",
    "*.html",
    "--cache-control",
    "no-cache",
    "--content-type",
    "text/html",
  ]);
  execute("aws", [
    "cloudfront",
    "create-invalidation",
    "--distribution-id",
    id,
    "--paths",
    "/*",
    "--region",
    "us-east-1",
  ]);
}

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(resolve(process.argv[1])).href
) {
  try {
    deployWeb();
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
