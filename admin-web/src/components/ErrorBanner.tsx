import { errorMessage } from '../lib/format'

export function ErrorBanner({ error }: { error: string | null }) {
  if (!error) return null
  return <div className="banner banner-error">{errorMessage(error)}</div>
}
