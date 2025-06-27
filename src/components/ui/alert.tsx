import * as React from "react";

export function Alert({ children, variant = "default" }: { children: React.ReactNode; variant?: "default" | "destructive" }) {
  return (
    <div
      className={`rounded-md border p-4 ${
        variant === "destructive"
          ? "border-red-400 bg-red-50 text-red-800"
          : "border-gray-200 bg-white text-gray-900"
      }`}
      role="alert"
    >
      {children}
    </div>
  );
}

export function AlertTitle({ children }: { children: React.ReactNode }) {
  return <div className="font-bold mb-1">{children}</div>;
}

export function AlertDescription({ children }: { children: React.ReactNode }) {
  return <div className="text-sm">{children}</div>;
}