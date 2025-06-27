"use client"

import * as React from "react"

export function Separator({ className = "", ...props }: React.HTMLAttributes<HTMLHRElement>) {
  return (
    <hr
      className={`border-t border-border my-4 ${className}`}
      {...props}
    />
  )
}