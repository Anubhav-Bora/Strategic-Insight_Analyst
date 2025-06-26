"use client";

import { Toaster as SonnerToaster, toast } from "sonner";
import React from "react";

// Optionally, you can create a custom hook for easy usage
export function useToast() {
  return toast;
}

// You can customize the Toaster here (position, theme, etc.)
export const Toaster = () => (
  <SonnerToaster
    position="top-right"
    richColors
    closeButton
    toastOptions={{
      style: {
        fontSize: "1rem",
        borderRadius: "8px",
        boxShadow: "0 2px 12px rgba(0,0,0,0.08)",
      },
    }}
  />
);