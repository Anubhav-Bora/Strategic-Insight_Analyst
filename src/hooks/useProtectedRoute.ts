import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { useAuth } from "./useAuth";

export function useProtectedRoute() {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && !user) {
      router.push("/login");
    }
  }, [user, loading, router]);

  return { user, loading };
}