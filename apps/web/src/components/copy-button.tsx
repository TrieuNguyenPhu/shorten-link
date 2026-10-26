"use client";

import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";

export function CopyButton({ value }: { value: string }) {
  const [result, setResult] = useState<"idle" | "copied" | "error">("idle");
  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setResult("copied");
    } catch {
      setResult("error");
    }
  }
  return (
    <div className="copy-control">
      <Button
        variant="outline"
        size="sm"
        onClick={copy}
        aria-label="Copy short link"
      >
        {result === "copied" ? (
          <Check aria-hidden="true" />
        ) : (
          <Copy aria-hidden="true" />
        )}
        {result === "copied" ? "Copied" : "Copy"}
      </Button>
      {result === "error" && (
        <span role="alert" className="copy-error">
          Select the link to copy it manually.
        </span>
      )}
    </div>
  );
}
