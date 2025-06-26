import { Skeleton } from "@/components/ui/skeleton";

export function DocumentListLoading() {
  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {[...Array(3)].map((_, i) => (
        <Skeleton key={i} className="h-32 w-full" />
      ))}
    </div>
  );
}

export function DocumentDetailLoading() {
  return (
    <div className="container mx-auto py-8">
      <div className="flex items-center justify-between">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-10 w-24" />
      </div>
      <Skeleton className="mt-8 h-96 w-full" />
    </div>
  );
}