import { cleanup, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, beforeEach, expect, it, vi } from "vitest"
const mocks = vi.hoisted(() => ({confirmEmailVerification:vi.fn(),requestEmailVerification:vi.fn()}))
vi.mock("next/navigation",()=>({useSearchParams:()=>new URLSearchParams("token=verification-test-token")}))
vi.mock("@/lib/backend-client",()=>({getCurrentUser:()=>null}))
vi.mock("@/services/user-management.service",()=>({usersService:mocks}))
import VerifyEmailPage from "./page"
beforeEach(()=>{mocks.confirmEmailVerification.mockResolvedValue({});mocks.requestEmailVerification.mockResolvedValue({})})
afterEach(()=>{cleanup();vi.clearAllMocks()})
it("requires user confirmation before consuming the verification token",async()=>{
 render(<VerifyEmailPage/>);
 expect(mocks.confirmEmailVerification).not.toHaveBeenCalled()
 await userEvent.click(screen.getByRole("button",{name:"Verify email"}))
 await waitFor(()=>expect(mocks.confirmEmailVerification).toHaveBeenCalledExactlyOnceWith("verification-test-token"))
 expect(await screen.findByText(/Your email ownership has been confirmed/)).toBeInTheDocument()
})
it("offers resend after an expired link with an account-neutral response",async()=>{
 mocks.confirmEmailVerification.mockRejectedValue(new Error("expired"))
 render(<VerifyEmailPage/>);
 await userEvent.click(screen.getByRole("button",{name:"Verify email"}))
 expect(await screen.findByText(/This link may be expired/)).toBeInTheDocument()
 await userEvent.type(screen.getByLabelText("Email address"),"person@example.test")
 await userEvent.click(screen.getByRole("button",{name:"Send verification link"}))
 expect(await screen.findByText(/If this account needs verification/)).toBeInTheDocument()
 expect(mocks.requestEmailVerification).toHaveBeenCalledWith("person@example.test")
})
