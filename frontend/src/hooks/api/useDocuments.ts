import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";

export function useDocuments() {
  return useQuery({
    queryKey: ["documents"],
    queryFn: async () => {
      const { data } = await api.get("/api/documents");
      return data;
    },
  });
}