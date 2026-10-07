"use client"

import { Suspense, useEffect, useState } from "react"
import Link from "next/link"
import { useSearchParams } from "next/navigation"
import { Loader2 } from "lucide-react"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { getCurrentUser } from "@/lib/backend-client"
import { usersService } from "@/services/user-management.service"

function VerifyEmailContent() {
  const token = useSearchParams().get("token")?.trim() ?? ""
  const [email, setEmail] = useState(() => String(getCurrentUser()?.email ?? ""))
  const [busy, setBusy] = useState(false)
  const [verified, setVerified] = useState(false)
  const [message, setMessage] = useState("")
  const [error, setError] = useState("")
  const [cooldown, setCooldown] = useState(0)

  useEffect(() => {
    if (cooldown <= 0) return
    const timer = setTimeout(() => setCooldown(value => value - 1), 1000)
    return () => clearTimeout(timer)
  }, [cooldown])

  async function perform(confirm: boolean) {
    if (busy || (!confirm && cooldown > 0)) return
    setBusy(true)
    setError("")
    setMessage("")
    try {
      if (confirm) {
        await usersService.confirmEmailVerification(token)
        setVerified(true)
      } else {
        await usersService.requestEmailVerification(email.trim())
        setCooldown(60)
        setMessage("If this account needs verification, a new link will be emailed. Check your inbox and spam folder.")
      }
    } catch {
      setError(confirm
        ? "This link may be expired or already used. Request a new link below."
        : "Unable to request a link. Please wait and try again.")
    } finally {
      setBusy(false)
    }
  }

  return (
    <Card className="mx-auto w-full max-w-md">
      <CardHeader><CardTitle>Email verification</CardTitle></CardHeader>
      <CardContent className="space-y-4">
        {verified ? (
          <Alert><AlertDescription>Your email ownership has been confirmed. Administrative account approval is separate.</AlertDescription></Alert>
        ) : (
          <>
            <p className="text-sm text-muted-foreground">Confirm your email address to keep your account contact details up to date.</p>
            {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
            {message && <Alert><AlertDescription>{message}</AlertDescription></Alert>}
            {token && (
              <>
                <Button disabled={busy} onClick={() => perform(true)} className="w-full">
                  {busy ? "Verifying…" : "Verify email"}
                </Button>
                <Button asChild variant="outline" className="w-full">
                  <a href={`mediguide://account/verify-email?token=${encodeURIComponent(token)}`}>Open in MediGuide app</a>
                </Button>
              </>
            )}
            <form className="space-y-3" onSubmit={event => { event.preventDefault(); void perform(false) }}>
              <Label htmlFor="verification-email">Email address</Label>
              <Input id="verification-email" type="email" autoComplete="email" value={email} onChange={event => setEmail(event.target.value)} required />
              <Button type="submit" variant="outline" disabled={busy || cooldown > 0} className="w-full">
                {cooldown > 0 ? `Resend in ${cooldown}s` : "Send verification link"}
              </Button>
            </form>
          </>
        )}
      </CardContent>
      <CardFooter>
        <Button asChild variant="ghost" className="w-full"><Link href="/login">Continue to sign in</Link></Button>
      </CardFooter>
    </Card>
  )
}

export default function VerifyEmailPage() {
  return <Suspense fallback={<Loader2 className="h-5 w-5 animate-spin" />}><VerifyEmailContent /></Suspense>
}
