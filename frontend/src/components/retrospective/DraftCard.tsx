import { PenLine } from 'lucide-react'

interface DraftCardProps {
  userName: string
  contentLength: number
  color: string
}

// Generate masked content (dashes) based on content length
const generateMaskedContent = (length: number): string => {
  // Create a visually interesting pattern with varying dash lengths
  const words: string[] = []
  let remaining = length

  while (remaining > 0) {
    // Random word length between 3 and 8 characters
    const wordLength = Math.min(Math.floor(Math.random() * 6) + 3, remaining)
    words.push('—'.repeat(wordLength))
    remaining -= wordLength + 1 // +1 for the space
  }

  return words.join(' ')
}

export default function DraftCard({ userName, contentLength, color }: DraftCardProps) {
  const maskedContent = generateMaskedContent(contentLength)

  return (
    <div
      className="bg-white rounded-lg border-2 border-dashed p-3 opacity-60 animate-pulse"
      style={{ borderColor: color }}
    >
      <div className="flex items-center gap-2 mb-2">
        <PenLine className="w-3 h-3 text-gray-400" />
        <span className="text-xs text-gray-500 italic">
          {userName} est en train d'écrire...
        </span>
      </div>
      <p className="text-sm text-gray-400 select-none" style={{ wordBreak: 'break-word' }}>
        {maskedContent}
      </p>
    </div>
  )
}
