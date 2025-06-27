import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";

export function useChatHistory(documentId: string) {
  return useQuery({
    queryKey: ["chatHistory", documentId],
    queryFn: async () => {
      const { data } = await api.get(`/api/documents/${documentId}/chat/history`);
      return data;
    },
  });
}