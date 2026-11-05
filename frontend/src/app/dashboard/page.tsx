import type { Metadata } from "next";
import { LinksWorkspace } from "@/features/links/LinksWorkspace";
export const metadata: Metadata = { title: "Dashboard" };
export default function DashboardPage() { return <LinksWorkspace />; }
