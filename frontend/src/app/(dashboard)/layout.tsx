"use client";

import Navigation from '@/components/Navigation'
import { useProtectedRoute } from '@/hooks/useProtectedRoute'

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  useProtectedRoute(); // This will redirect to /login if not authenticated

  return (
    <>
      <Navigation />
      {children}
    </>
  )
}