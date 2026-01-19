import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Users, ChevronDown, ChevronRight, Calendar, Shield, User } from 'lucide-react'
import clsx from 'clsx'
import { adminApi } from '../api/client'
import type { TeamWithMemberCount, TeamMember } from '../types'

function TeamMembersList({ teamId }: { teamId: string }) {
  const { data: members, isLoading } = useQuery({
    queryKey: ['admin', 'teams', teamId, 'members'],
    queryFn: () => adminApi.getTeamMembers(teamId),
  })

  if (isLoading) {
    return (
      <div className="py-4 px-6 flex items-center justify-center">
        <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-primary-600" />
      </div>
    )
  }

  if (!members || members.length === 0) {
    return (
      <div className="py-4 px-6 text-sm text-gray-500 italic">
        Aucun membre dans cette équipe
      </div>
    )
  }

  const getRoleBadge = (role: string) => {
    switch (role) {
      case 'admin':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium bg-purple-100 text-purple-700 rounded">
            <Shield className="w-3 h-3" />
            Admin
          </span>
        )
      case 'member':
      default:
        return (
          <span className="px-2 py-0.5 text-xs font-medium bg-gray-100 text-gray-700 rounded">
            Membre
          </span>
        )
    }
  }

  return (
    <div className="py-2 px-6 bg-gray-50 border-t border-gray-100">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
        {members.map((member: TeamMember) => (
          <div
            key={member.id}
            className="flex items-center gap-3 p-3 bg-white rounded-lg border border-gray-200"
          >
            {member.user?.avatarUrl ? (
              <img
                src={member.user.avatarUrl}
                alt={member.user.displayName}
                className="w-8 h-8 rounded-full"
              />
            ) : (
              <div className="w-8 h-8 rounded-full bg-primary-100 flex items-center justify-center">
                <span className="text-primary-600 font-medium text-sm">
                  {member.user?.displayName?.charAt(0).toUpperCase() || '?'}
                </span>
              </div>
            )}
            <div className="flex-1 min-w-0">
              <div className="text-sm font-medium text-gray-900 truncate">
                {member.user?.displayName || 'Utilisateur inconnu'}
              </div>
              <div className="flex items-center gap-2 mt-1">
                {getRoleBadge(member.role)}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

export default function TeamsAdminPage() {
  const [expandedTeams, setExpandedTeams] = useState<Set<string>>(new Set())

  const { data: teams, isLoading } = useQuery({
    queryKey: ['admin', 'teams'],
    queryFn: () => adminApi.listTeams(),
  })

  const toggleTeam = (teamId: string) => {
    setExpandedTeams((prev) => {
      const next = new Set(prev)
      if (next.has(teamId)) {
        next.delete(teamId)
      } else {
        next.add(teamId)
      }
      return next
    })
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center gap-4">
          <div className="p-3 bg-primary-100 rounded-lg">
            <Users className="w-6 h-6 text-primary-600" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Équipes</h1>
            <p className="text-gray-500">{teams?.length || 0} équipe{(teams?.length || 0) > 1 ? 's' : ''} créée{(teams?.length || 0) > 1 ? 's' : ''}</p>
          </div>
        </div>
      </div>

      {/* Teams List */}
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        {teams && teams.length > 0 ? (
          <div className="divide-y divide-gray-200">
            {teams.map((team: TeamWithMemberCount) => (
              <div key={team.id}>
                <div
                  className={clsx(
                    'flex items-center gap-4 px-6 py-4 cursor-pointer hover:bg-gray-50 transition-colors',
                    expandedTeams.has(team.id) && 'bg-gray-50'
                  )}
                  onClick={() => toggleTeam(team.id)}
                >
                  <button className="text-gray-400">
                    {expandedTeams.has(team.id) ? (
                      <ChevronDown className="w-5 h-5" />
                    ) : (
                      <ChevronRight className="w-5 h-5" />
                    )}
                  </button>

                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-3">
                      <h3 className="text-sm font-medium text-gray-900">
                        {team.name}
                      </h3>
                      <span className="text-xs text-gray-500">@{team.slug}</span>
                    </div>
                    {team.description && (
                      <p className="mt-1 text-sm text-gray-500 truncate">
                        {team.description}
                      </p>
                    )}
                  </div>

                  <div className="flex items-center gap-6 text-sm text-gray-500">
                    <div className="flex items-center gap-2">
                      <User className="w-4 h-4" />
                      <span>{team.memberCount} membre{team.memberCount > 1 ? 's' : ''}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Calendar className="w-4 h-4" />
                      <span>{new Date(team.createdAt).toLocaleDateString('fr-FR')}</span>
                    </div>
                  </div>
                </div>

                {expandedTeams.has(team.id) && (
                  <TeamMembersList teamId={team.id} />
                )}
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <Users className="w-12 h-12 text-gray-400 mx-auto" />
            <h3 className="mt-4 text-lg font-medium text-gray-900">Aucune équipe</h3>
            <p className="mt-2 text-gray-600">
              Aucune équipe n'a encore été créée.
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
