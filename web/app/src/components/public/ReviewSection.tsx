import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import {
  fetchReviews,
  fetchReviewSummary,
  submitReview,
  deleteReview,
} from "@/api/client"
import type { Review, ReviewSummary } from "@/api/types"
import { useAuth } from "@/hooks/use-auth"
import { StarRating } from "./StarRating"

interface ReviewSectionProps {
  agentId: string
}

export function ReviewSection({ agentId }: ReviewSectionProps) {
  const { t } = useTranslation()
  const { user, accessToken } = useAuth()
  const [reviews, setReviews] = useState<Review[]>([])
  const [summary, setSummary] = useState<ReviewSummary | null>(null)
  const [loading, setLoading] = useState(true)

  const [newRating, setNewRating] = useState(0)
  const [newComment, setNewComment] = useState("")
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState("")

  const load = () => {
    setLoading(true)
    Promise.all([
      fetchReviews(agentId),
      fetchReviewSummary(agentId),
    ])
      .then(([revRes, sum]) => {
        setReviews(revRes.reviews ?? [])
        setSummary(sum)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
  }, [agentId])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!accessToken || newRating === 0) return
    setSubmitting(true)
    setError("")
    try {
      await submitReview(agentId, newRating, newComment, accessToken)
      setNewRating(0)
      setNewComment("")
      load()
    } catch (err: any) {
      setError(err.message)
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async () => {
    if (!accessToken) return
    try {
      await deleteReview(agentId, accessToken)
      load()
    } catch {}
  }

  const myReview = reviews.find((r) => r.user_id === user?.id)

  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <h2 className="mb-4 text-sm font-semibold">{t('review.title')}</h2>

      {/* Summary */}
      {summary && summary.total_reviews > 0 && (
        <div className="mb-4 flex items-center gap-4">
          <div className="text-center">
            <div className="text-3xl font-bold">
              {summary.average_rating.toFixed(1)}
            </div>
            <StarRating rating={Math.round(summary.average_rating)} size="sm" />
            <div className="mt-1 text-xs text-muted-foreground">
              {t('review.reviewCount', { count: summary.total_reviews })}
            </div>
          </div>
          <div className="flex-1 space-y-1">
            {[5, 4, 3, 2, 1].map((star, idx) => (
              <div key={star} className="flex items-center gap-2 text-xs">
                <span className="w-3 text-right">{star}</span>
                <div className="h-2 flex-1 rounded-full bg-zinc-800">
                  <div
                    className="h-2 rounded-full bg-yellow-400"
                    style={{
                      width: `${
                        summary.total_reviews > 0 && summary.distribution
                          ? ((summary.distribution[4 - idx] ?? 0) / summary.total_reviews) * 100
                          : 0
                      }%`,
                    }}
                  />
                </div>
                <span className="w-6 text-right text-muted-foreground">
                  {summary.distribution?.[4 - idx] ?? 0}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Submit form */}
      {user && !myReview && (
        <form onSubmit={handleSubmit} className="mb-4 space-y-3 border-b border-border pb-4">
          <div>
            <label className="mb-1 block text-xs text-muted-foreground">{t('review.yourRating')}</label>
            <StarRating rating={newRating} onChange={setNewRating} />
          </div>
          <textarea
            placeholder={t('review.writeReview')}
            value={newComment}
            onChange={(e) => setNewComment(e.target.value)}
            className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            rows={2}
          />
          {error && <p className="text-xs text-destructive">{error}</p>}
          <button
            type="submit"
            disabled={submitting || newRating === 0}
            className="rounded-lg bg-primary px-4 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {submitting ? t('review.submitting') : t('review.submitReview')}
          </button>
        </form>
      )}

      {/* Review list */}
      {loading ? (
        <p className="text-xs text-muted-foreground">{t('review.loadingReviews')}</p>
      ) : reviews.length === 0 ? (
        <p className="text-xs text-muted-foreground">{t('review.noReviews')}</p>
      ) : (
        <div className="space-y-3">
          {reviews.map((rev) => (
            <div key={rev.id} className="border-b border-border/50 pb-3 last:border-0">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <StarRating rating={rev.rating} size="sm" />
                  <span className="text-xs text-muted-foreground">
                    {new Date(rev.created_at).toLocaleDateString()}
                  </span>
                </div>
                {rev.user_id === user?.id && (
                  <button
                    onClick={handleDelete}
                    className="text-xs text-destructive hover:underline"
                  >
                    {t('review.delete')}
                  </button>
                )}
              </div>
              {rev.comment && (
                <p className="mt-1 text-sm text-foreground">{rev.comment}</p>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
