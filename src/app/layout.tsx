import type React from "react"
import { Inter } from "next/font/google"
import "../globals.css"
import { AuthProvider } from "@/context/AuthContext"
import { QueryProvider } from "@/providers/QueryProvider"
import { ToastProvider } from "@/providers/ToastProvider"
import { ThemeProvider } from "@/components/theme-provider"

const inter = Inter({ subsets: ["latin"] })

export const metadata = {
  title: "Strategic Insight - AI Document Analysis",
  description: "Transform your documents into strategic insights with AI-powered analysis",
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={inter.className}>
        <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
          <QueryProvider>
            <AuthProvider>
              <ToastProvider>{children}</ToastProvider>
            </AuthProvider>
          </QueryProvider>
        </ThemeProvider>
      </body>
    </html>
  )
}
