import { existsSync, readFileSync } from "node:fs";
import { registerHooks } from "node:module";
import { fileURLToPath } from "node:url";
import ts from "typescript";

const sourceRoot = new URL("../../src/", import.meta.url);

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "next/image") {
      const target = nextResolve("next/image.js", context).url;
      return nextResolve(`data:text/javascript,${encodeURIComponent(`import image from ${JSON.stringify(target)}; export default image.default;`)}`, context);
    }
    if (specifier === "next/link" || specifier === "next/navigation") specifier += ".js";
    let target;
    if (specifier.startsWith("@/")) target = new URL(specifier.slice(2), sourceRoot);
    else if (specifier.startsWith(".") && context.parentURL?.startsWith(sourceRoot.href)) {
      target = new URL(specifier, context.parentURL);
    }
    if (target && !/\.[a-z]+$/.test(target.pathname)) {
      for (const extension of [".tsx", ".ts"]) {
        const candidate = new URL(target.href + extension);
        if (existsSync(candidate)) return nextResolve(candidate.href, context);
      }
    }
    return nextResolve(specifier, context);
  },
  load(url, context, nextLoad) {
    if (!url.startsWith(sourceRoot.href) || !url.endsWith(".tsx")) return nextLoad(url, context);
    const { outputText } = ts.transpileModule(readFileSync(new URL(url), "utf8"), {
      fileName: fileURLToPath(url),
      compilerOptions: { jsx: ts.JsxEmit.ReactJSX, module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
    });
    return { format: "module", shortCircuit: true, source: outputText };
  },
});
