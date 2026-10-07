import { afterEach, expect, it, vi } from "vitest"
import type { User } from "./columns"
const service = vi.hoisted(() => ({verify:vi.fn(),requestEmailVerification:vi.fn()}))
vi.mock("@/services/user-management.service",()=>({usersService:service}))
vi.mock("@/lib/toast",()=>({showToast:{info:vi.fn(),success:vi.fn(),error:vi.fn()}}))
import { createUserRowActions } from "./user-actions"
const user = {id:"user-id",name:"User",email:"person@example.test",verified:false,email_verified:false} as User
afterEach(()=>vi.clearAllMocks())
it("keeps administrative approval separate from email verification",async()=>{
 const actions=createUserRowActions(vi.fn())
 const approval=actions.find(action=>action.id==="send-verification")!
 expect(approval.label).toBe("Approve Account")
 await approval.onClick(user)
 expect(service.verify).toHaveBeenCalledWith("user-id")
 expect(service.requestEmailVerification).not.toHaveBeenCalled()
})
it("resends email verification without approving an account",async()=>{
 const resend=createUserRowActions(vi.fn()).find(action=>action.id==="resend-email-verification")!
 await resend.onClick(user)
 expect(service.requestEmailVerification).toHaveBeenCalledWith("person@example.test")
 expect(service.verify).not.toHaveBeenCalled()
 expect(resend.disabled?.({...user,email_verified:true})).toBe(true)
})
