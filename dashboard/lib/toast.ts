import { toast } from "sonner"
import { isSessionExpiryToastWindow } from "@/lib/backend-client"

export const showToast = {
  success: (message: string, description?: string) => {
    toast.success(message, {
      description,
    })
  },
  error: (message: string, description?: string) => {
    // Swallowed: a real session expiry already surfaces its own single
    // "Session Expired" toast (see AuthGuard), and every request failing at
    // once because of it would otherwise flood the user with duplicates.
    if (isSessionExpiryToastWindow()) return
    toast.error(message, {
      description,
    })
  },
  warning: (message: string, description?: string) => {
    if (isSessionExpiryToastWindow()) return
    toast.warning(message, {
      description,
    })
  },
  info: (message: string, description?: string) => {
    toast.info(message, {
      description,
    })
  },
  message: (message: string, description?: string) => {
    toast.message(message, {
      description,
    })
  },
  loading: (message: string) => {
    return toast.loading(message)
  },
  promise: <T,>(
    promise: Promise<T>,
    {
      loading,
      success,
      error,
    }: {
      loading: string
      success: string | ((data: T) => string)
      error: string | ((error: unknown) => string)
    }
  ) => {
    return toast.promise(promise, {
      loading,
      success,
      error,
    })
  },
}