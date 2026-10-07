"use client"

import { useEffect, useState } from "react"
import { getBackendClient } from "@/lib/backend-client"
import { PageHeader } from "@/components/ui/page-header"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription } from "@/components/ui/alert"

type Summary = {
  since: string
  events: { event: string; count: number }[]
  email_queue: { status: string; count: number }[]
}

const labels: Record<string, string> = {
  "user.account_created": "Accounts created",
  "user.password_reset_requested": "Reset links requested",
  "user.password_reset_completed": "Passwords reset",
  "user.email_verification_requested": "Verification links requested",
  "user.email_verified": "Emails verified",
  "user.verified": "Accounts approved by administrators",
  "user.account_email_sent": "Emails accepted by provider",
  "user.account_email_send_failed": "Email send attempts failed",
  "user.account_email_failed": "Emails exhausted retries",
  "user.account_email_cancelled": "Expired or superseded emails cancelled",
}

export default function AccountAnalyticsPage() {
  const [data, setData] = useState<Summary | null>(null)
  const [error, setError] = useState(false)
  const [loading, setLoading] = useState(true)
  const [reload, setReload] = useState(0)

  useEffect(() => {
    let active = true
    setLoading(true)
    setError(false)
    getBackendClient().send<Summary>("/api/v2/analytics/accounts", { query: { days: 30 } })
      .then(result => { if (active) setData(result) })
      .catch(() => { if (active) setError(true) })
      .finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [reload])

  return (
    <div className="space-y-6">
      <PageHeader title="Account lifecycle" description="Registration, recovery and verification over the last 30 days." />
      <Button onClick={() => setReload(value => value + 1)} disabled={loading}>
        {loading ? "Loading…" : "Refresh"}
      </Button>
      {error && (
        <Alert variant="destructive"><AlertDescription>Unable to load account analytics. Check your permissions and try again.</AlertDescription></Alert>
      )}
      {!loading && !error && data && (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {Object.entries(labels).map(([key, label]) => (
              <Card key={key}>
                <CardHeader><CardTitle className="text-sm">{label}</CardTitle></CardHeader>
                <CardContent><p className="text-3xl font-semibold">{data.events.find(event => event.event === key)?.count ?? 0}</p></CardContent>
              </Card>
            ))}
          </div>
          <Card>
            <CardHeader><CardTitle>Email queue</CardTitle></CardHeader>
            <CardContent className="space-y-2">
              {data.email_queue.length ? data.email_queue.map(delivery => (
                <p key={delivery.status} className="flex justify-between">
                  <span className="capitalize">{delivery.status}</span><span>{delivery.count}</span>
                </p>
              )) : <p>No account emails have been queued yet.</p>}
            </CardContent>
          </Card>
          <p className="text-sm text-muted-foreground">
            Provider acceptance does not confirm inbox delivery. Counts cover events recorded after lifecycle tracking was enabled; older accounts are not counted as new registrations.
          </p>
        </>
      )}
    </div>
  )
}
