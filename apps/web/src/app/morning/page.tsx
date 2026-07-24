"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

export default function MorningPage() {
  const router = useRouter();
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/api/morning")
      .then((res) => {
        if (res.status === 401) {
          router.push("/login");
          return null;
        }
        return res.json();
      })
      .then((data) => {
        if (data) setMessage(data.message);
      })
      .catch(() => setMessage("Failed to fetch"))
      .finally(() => setLoading(false));
  }, [router]);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-50">
        <p className="text-zinc-500">Loading...</p>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-50">
      <div className="flex flex-col items-center gap-4 rounded-xl bg-white p-8 shadow-sm">
        <h1 className="text-2xl font-semibold text-zinc-900">
          {message || "Good Morning!"}
        </h1>
        <button
          onClick={() => router.push("/login")}
          className="rounded-lg bg-zinc-100 px-4 py-2 text-sm text-zinc-600 hover:bg-zinc-200"
        >
          Sign out
        </button>
      </div>
    </div>
  );
}