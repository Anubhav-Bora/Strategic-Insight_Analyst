import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";

export function useDocument(id: string) {
  return useQuery({
    queryKey: ["document", id],
    queryFn: async () => {
      const { data } = await api.get(`/api/documents/${id}`);
      return data;
    },
  });
}