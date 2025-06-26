"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { signOut } from "firebase/auth";
import { auth } from "@/lib/firebase";
import { useRouter } from "next/navigation";
import { useAuthContext } from "@/context/AuthContext";

export default function Navigation() {
  const { user } = useAuthContext();
  const router = useRouter();

  async function handleSignOut() {
    try {
      await signOut(auth);
      router.push("/login");
    } catch (error) {
      console.error("Error signing out:", error);
    }
  }

  return (
    <header className="border-b">
      <div className="container flex h-16 items-center justify-between">
        <Link href="/" className="font-bold">
          Strategic Insight Analyst
        </Link>
        {user && (
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-600">{user.email}</span>
            <Button variant="outline" size="sm" onClick={handleSignOut}>
              Sign out
            </Button>
          </div>
        )}
      </div>
    </header>
  );
}