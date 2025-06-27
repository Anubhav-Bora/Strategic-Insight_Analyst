"use client";

import Navigation from '@/components/Navigation'
import { useProtectedRoute } from '@/hooks/useProtectedRoute'
import "../../globals.css"

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  useProtectedRoute(); // This will redirect to /login if not authenticated

  return (
    <>
      <Navigation />
      <div className="pt-16">
        {children}
      </div>
    </>
  )
}