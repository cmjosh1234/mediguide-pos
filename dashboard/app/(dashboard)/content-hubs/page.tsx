"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Plus, LayoutGrid, SearchX } from "lucide-react";
import { EmptyState } from "@/components/ui/empty-state";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { PageHeader } from "@/components/ui/page-header";
import { hasBackendPermission } from "@/lib/backend-client";
import { showToast } from "@/lib/toast";
import { ContentHub, contentHubService } from "@/services/content-hubs.service";

export default function ContentHubsPage() {
  const router = useRouter();
  const canManage = hasBackendPermission("content_hub.manage");
  const [rows, setRows] = React.useState<ContentHub[]>([]);
  const [search, setSearch] = React.useState("");
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState("");
  const [revision, setRevision] = React.useState(0);
  React.useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(() => {
      setLoading(true);
      setError("");
      void contentHubService
        .list({ search })
        .then((value) => { if (!cancelled) setRows(value.items || []) })
        .catch((error) => {
          if (cancelled) return;
          setError(error instanceof Error ? error.message : "Unable to load hubs.");
          showToast.error(
            "Hubs unavailable",
            error instanceof Error ? error.message : "Try again.",
          );
        })
        .finally(() => { if (!cancelled) setLoading(false) });
    }, 200);
    return () => { cancelled = true; clearTimeout(timer) };
  }, [search, revision]);
  return (
    <div className="space-y-6">
      <PageHeader
        title="Content hubs"
        description="Curate disease and outbreak resources into API-managed pillars for web and mobile."
      />
      <div className="flex gap-2">
        <Input
          value={search}
          onChange={(event) => { setLoading(true); setSearch(event.target.value) }}
          placeholder="Search hubs"
        />
        {canManage ? <Button asChild>
          <Link href="/content-hubs/new">
            <Plus className="mr-2 h-4 w-4" />
            New hub
          </Link>
        </Button> : null}
      </div>
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        {!loading && !error && rows.map((row) => (
          <Card key={row.id}>
            <CardHeader className="pb-2">
              <div className="flex justify-between gap-2">
                <CardTitle className="text-base">{row.name}</CardTitle>
                <Badge variant="outline">{row.status}</Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <p className="text-muted-foreground">
                {row.description || "No description"}
              </p>
              <p>
                {row.diseases?.map((value) => value.name).join(", ") ||
                  "General hub"}
              </p>
              <Button variant="outline" size="sm" asChild>
                <Link href={`/content-hubs/${row.id}`}>
                  {canManage ? "Manage hub" : "View hub"}
                </Link>
              </Button>
            </CardContent>
          </Card>
        ))}
      </div>
      {loading ? <p role="status" className="py-12 text-center text-muted-foreground">Loading content hubs…</p> : error ? <div role="alert" className="rounded-md border p-6"><p>{error}</p><Button className="mt-3" variant="outline" onClick={() => { setLoading(true); setRevision((value) => value + 1) }}>Try again</Button></div> : rows.length === 0 ? <EmptyState
        icon={search.trim() ? SearchX : LayoutGrid}
        title={search.trim() ? "No matching content hubs" : "No content hubs yet"}
        description={search.trim() ? "Try another name or clear your search." : "Create a hub to organize approved disease and outbreak resources."}
        action={search.trim() ? { label: "Clear search", onClick: () => { setLoading(true); setSearch("") } } : canManage ? { label: "New hub", onClick: () => router.push("/content-hubs/new") } : { label: "Refresh", onClick: () => { setLoading(true); setRevision((value) => value + 1) } }}
      /> : null}
    </div>
  );
}
