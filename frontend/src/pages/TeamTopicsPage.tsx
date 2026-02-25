import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { teamsApi } from '../api/client'
import {
  Coffee,
  Users,
  Clock,
  MessageSquare,
  Calendar,
  Timer,
  Plus,
  ChevronDown,
  ChevronRight,
} from 'lucide-react'
import type { DiscussedTopic } from '../types'

function formatDuration(seconds: number): string {
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  if (mins === 0) return `${secs}s`
  return secs > 0 ? `${mins}min ${secs}s` : `${mins}min`
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('fr-FR', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  })
}

interface GroupedTopics {
  sessionId: string
  sessionName: string
  date: string
  topics: DiscussedTopic[]
  totalDuration: number
}

function groupTopicsBySession(topics: DiscussedTopic[]): GroupedTopics[] {
  const groups = new Map<string, GroupedTopics>()

  for (const topic of topics) {
    if (!groups.has(topic.sessionId)) {
      groups.set(topic.sessionId, {
        sessionId: topic.sessionId,
        sessionName: topic.sessionName,
        date: topic.discussedAt,
        topics: [],
        totalDuration: 0,
      })
    }
    const group = groups.get(topic.sessionId)!
    group.topics.push(topic)
    group.totalDuration += topic.totalDiscussionSeconds
  }

  // Sort by date descending
  return Array.from(groups.values()).sort(
    (a, b) => new Date(b.date).getTime() - new Date(a.date).getTime()
  )
}

function getTrigram(name: string): string {
  if (!name) return '???'
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 3) return parts.slice(0, 3).map(p => p[0]?.toUpperCase() || '').join('')
  if (parts.length === 2) return (parts[0][0] + parts[1].slice(0, 2)).toUpperCase()
  return name.slice(0, 3).toUpperCase()
}

function TopicCard({ topic }: { topic: DiscussedTopic }) {
  return (
    <div className="flex items-start gap-3 p-3 bg-white rounded-lg border border-gray-100 hover:border-amber-200 transition-colors">
      <div className="flex-shrink-0 mt-0.5 inline-flex items-center justify-center w-7 h-7 rounded-full bg-amber-100 text-amber-700 text-xs font-bold">
        {getTrigram(topic.authorName)}
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm text-gray-900">{topic.content}</p>
        <div className="flex items-center gap-3 mt-2 text-xs text-gray-500">
          <span className="flex items-center gap-1">
            {topic.authorName}
          </span>
          <span className="flex items-center gap-1">
            <Timer className="w-3 h-3" />
            {formatDuration(topic.totalDiscussionSeconds)}
          </span>
          {topic.extensionCount > 0 && (
            <span className="flex items-center gap-1 text-amber-600">
              <Plus className="w-3 h-3" />
              {topic.extensionCount} extension{topic.extensionCount > 1 ? 's' : ''}
            </span>
          )}
        </div>
      </div>
      <div className="flex-shrink-0 text-xs text-gray-400 font-mono">
        #{topic.discussionOrder}
      </div>
    </div>
  )
}

function SessionGroup({ group }: { group: GroupedTopics }) {
  const [expanded, setExpanded] = useState(true)

  return (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center justify-between p-4 hover:bg-gray-50 transition-colors"
      >
        <div className="flex items-center gap-3">
          <Coffee className="w-5 h-5 text-amber-500" />
          <div className="text-left">
            <h3 className="font-medium text-gray-900">{group.sessionName}</h3>
            <p className="text-xs text-gray-500 flex items-center gap-2 mt-0.5">
              <Calendar className="w-3 h-3" />
              {formatDate(group.date)}
              <span className="text-gray-300">|</span>
              <Clock className="w-3 h-3" />
              {formatDuration(group.totalDuration)} au total
              <span className="text-gray-300">|</span>
              {group.topics.length} sujet{group.topics.length > 1 ? 's' : ''}
            </p>
          </div>
        </div>
        {expanded ? (
          <ChevronDown className="w-5 h-5 text-gray-400" />
        ) : (
          <ChevronRight className="w-5 h-5 text-gray-400" />
        )}
      </button>

      {expanded && (
        <div className="px-4 pb-4 space-y-2">
          {group.topics
            .sort((a, b) => a.discussionOrder - b.discussionOrder)
            .map(topic => (
              <TopicCard key={topic.id} topic={topic} />
            ))}
        </div>
      )}
    </div>
  )
}

export default function TeamTopicsPage() {
  const { teamId } = useParams<{ teamId: string }>()

  const { data: team, isLoading: teamLoading } = useQuery({
    queryKey: ['team', teamId],
    queryFn: () => teamsApi.get(teamId!),
    enabled: !!teamId,
  })

  const { data: topics, isLoading: topicsLoading } = useQuery({
    queryKey: ['teamTopics', teamId],
    queryFn: () => teamsApi.getTopics(teamId!),
    enabled: !!teamId,
  })

  if (teamLoading || topicsLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-amber-600"></div>
      </div>
    )
  }

  if (!team) {
    return <div>Team not found</div>
  }

  const grouped = groupTopicsBySession(topics || [])
  const totalTopics = (topics || []).length
  const totalSessions = grouped.length
  const totalDuration = (topics || []).reduce((sum, t) => sum + t.totalDiscussionSeconds, 0)

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link
              to={`/teams/${teamId}`}
              className="p-3 bg-amber-100 rounded-lg hover:bg-amber-200 transition-colors"
            >
              <Users className="w-6 h-6 text-amber-600" />
            </Link>
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Sujets discutés</h1>
              <p className="text-gray-500">@{team.name} - Lean Coffee</p>
            </div>
          </div>
        </div>

        {/* Stats */}
        <div className="mt-6 grid grid-cols-3 gap-4">
          <div className="bg-amber-50 rounded-lg p-4">
            <div className="flex items-center gap-2 text-amber-600">
              <Coffee className="w-4 h-4" />
              <span className="text-sm">Sessions</span>
            </div>
            <p className="text-2xl font-bold text-gray-900 mt-1">{totalSessions}</p>
          </div>
          <div className="bg-amber-50 rounded-lg p-4">
            <div className="flex items-center gap-2 text-amber-600">
              <MessageSquare className="w-4 h-4" />
              <span className="text-sm">Sujets</span>
            </div>
            <p className="text-2xl font-bold text-gray-900 mt-1">{totalTopics}</p>
          </div>
          <div className="bg-amber-50 rounded-lg p-4">
            <div className="flex items-center gap-2 text-amber-600">
              <Clock className="w-4 h-4" />
              <span className="text-sm">Temps total</span>
            </div>
            <p className="text-2xl font-bold text-gray-900 mt-1">{formatDuration(totalDuration)}</p>
          </div>
        </div>
      </div>

      {/* Topics grouped by session */}
      {grouped.length > 0 ? (
        <div className="space-y-4">
          {grouped.map(group => (
            <SessionGroup key={group.sessionId} group={group} />
          ))}
        </div>
      ) : (
        <div className="text-center py-12 bg-white rounded-lg border border-gray-200">
          <Coffee className="w-12 h-12 text-gray-400 mx-auto" />
          <h3 className="mt-4 text-lg font-medium text-gray-900">Aucun sujet discuté</h3>
          <p className="mt-2 text-gray-600">
            Les sujets des Lean Coffees terminés apparaîtront ici
          </p>
          <Link
            to={`/teams/${teamId}`}
            className="mt-4 inline-flex items-center gap-2 px-4 py-2 bg-amber-600 text-white rounded-lg hover:bg-amber-700"
          >
            Retour à l'équipe
          </Link>
        </div>
      )}
    </div>
  )
}
