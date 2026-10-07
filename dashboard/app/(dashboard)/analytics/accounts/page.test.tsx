import { cleanup, render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, expect, it, vi } from "vitest"
const send=vi.hoisted(()=>vi.fn())
vi.mock("@/lib/backend-client",()=>({getBackendClient:()=>({send})}))
import AccountAnalyticsPage from "./page"
afterEach(()=>{cleanup();vi.clearAllMocks()})
it("shows an empty queue and zero counts without inventing historical registrations",async()=>{
 send.mockResolvedValue({since:"2026-10-01",events:[],email_queue:[]})
 render(<AccountAnalyticsPage/>);
 expect(await screen.findByText("No account emails have been queued yet.")).toBeInTheDocument()
 expect(screen.getAllByText("0")).toHaveLength(10)
 expect(send).toHaveBeenCalledWith("/api/v2/analytics/accounts",{query:{days:30}})
})
it("allows recovery from analytics errors",async()=>{
 send.mockRejectedValueOnce(new Error("unavailable")).mockResolvedValueOnce({since:"2026-10-01",events:[],email_queue:[{status:"pending",count:2}]})
 render(<AccountAnalyticsPage/>);
 expect(await screen.findByText(/Unable to load account analytics/)).toBeInTheDocument()
 await userEvent.click(screen.getByRole("button",{name:"Refresh"}))
 expect(await screen.findByText("pending")).toBeInTheDocument()
})
