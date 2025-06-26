export interface User {
    id: string;
    email: string;
    created_at: string;
  }
  
  export interface Document {
    id: string;
    user_id: string;
    file_name: string;
    storage_path: string;
    uploaded_at: string;
  }
  
  export interface DocumentChunk {
    id: string;
    document_id: string;
    chunk_index: number;
    content: string;
    created_at: string;
  }
  
  export interface ChatMessage {
    id: string;
    document_id: string;
    user_id: string;
    message_type: "user" | "ai";
    message_content: string;
    timestamp: string;
  }