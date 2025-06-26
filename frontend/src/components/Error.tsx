import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import Link from "next/link";

export function ErrorMessage({ message }: { message: string }) {
  return (
    <Alert variant="destructive">
      <AlertCircle className="h-4 w-4" />
      <AlertTitle>Error</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}

export function NotFoundMessage({ resource }: { resource: string }) {
  return (
    <div className="flex flex-col items-center justify-center space-y-4">
      <AlertCircle className="h-12 w-12 text-gray-400" />
      <h3 className="text-lg font-medium">{resource} not found</h3>
      <Link href="/">
        <Button>Go back</Button>
      </Link>
    </div>
  );
}