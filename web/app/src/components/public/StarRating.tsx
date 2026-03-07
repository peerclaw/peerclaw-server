import { Star } from "lucide-react"

interface StarRatingProps {
  rating: number
  onChange?: (rating: number) => void
  size?: "sm" | "md"
}

export function StarRating({ rating, onChange, size = "md" }: StarRatingProps) {
  const starSize = size === "sm" ? "size-4" : "size-5"

  return (
    <div className="inline-flex gap-0.5">
      {[1, 2, 3, 4, 5].map((star) => (
        <button
          key={star}
          type="button"
          disabled={!onChange}
          onClick={() => onChange?.(star)}
          className={onChange ? "cursor-pointer hover:scale-110 transition-transform" : "cursor-default"}
        >
          <Star
            className={`${starSize} ${
              star <= rating
                ? "fill-yellow-400 text-yellow-400"
                : "text-zinc-600"
            }`}
          />
        </button>
      ))}
    </div>
  )
}
