import Image from "next/image";

export function LinkIcon({ className = "h-10 w-10" }: { className?: string }) {
  return <Image alt="" aria-hidden="true" className={className} height={48} src="/icon.svg" width={48} />;
}
