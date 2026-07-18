"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"

const navItems = [
  { href: "/", label: "Overview" },
  { href: "/analytics", label: "Usage Analytics" },
  { href: "/limits", label: "Limit Management" },
]

export default function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()

  return (
    <div className="flex min-h-screen">
      <aside className="flex w-64 flex-col border-r bg-background px-4 py-6">
        <div className="mb-8 px-3 text-lg font-semibold tracking-tight">
          Sentinel
        </div>
        <nav className="flex flex-col gap-1">
          {navItems.map((item) => (
            <Link key={item.href} href={item.href}>
              <Button
                variant="ghost"
                className={cn(
                  "w-full justify-start",
                  pathname === item.href &&
                    "bg-accent text-accent-foreground font-medium"
                )}
              >
                {item.label}
              </Button>
            </Link>
          ))}
        </nav>
      </aside>
      <main className="flex-1 p-8">{children}</main>
    </div>
  )
}
