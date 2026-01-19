import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeft, Users, User, Star, Cloud, TrendingUp, Percent } from 'lucide-react'
import clsx from 'clsx'

import { statsApi, teamsApi } from '../api/client'
import StatCard from '../components/stats/StatCard'
import RotiDistributionChart from '../components/stats/RotiDistributionChart'
import MoodDistributionChart from '../components/stats/MoodDistributionChart'
import RotiEvolutionChart from '../components/stats/RotiEvolutionChart'
import MoodEvolutionChart from '../components/stats/MoodEvolutionChart'
import type { TeamMember, MoodWeather } from '../types'

type Tab = 'team' | 'individual'
type PeriodFilter = 'all' | '5' | '10' | '20'

const PERIOD_OPTIONS: { value: PeriodFilter; label: string }[] = [
  { value: 'all', label: 'Toutes les rétros' },
  { value: '5', label: '5 dernières' },
  { value: '10', label: '10 dernières' },
  { value: '20', label: '20 dernières' },
]

const MOOD_LABELS: Record<MoodWeather, string> = {
  sunny: 'Ensoleillé',
  partly_cloudy: 'Partiellement nuageux',
  cloudy: 'Nuageux',
  rainy: 'Pluvieux',
  stormy: 'Orageux',
}

export default function TeamStatsPage() {
  const { teamId } = useParams<{ teamId: string }>()
  const [activeTab, setActiveTab] = useState<Tab>('team')
  const [periodFilter, setPeriodFilter] = useState<PeriodFilter>('all')
  const [selectedMemberId, setSelectedMemberId] = useState<string | null>(null)

  const limit = periodFilter === 'all' ? undefined : parseInt(periodFilter)

  const { data: team, isLoading: teamLoading } = useQuery({
    queryKey: ['team', teamId],
    queryFn: () => teamsApi.get(teamId!),
    enabled: !!teamId,
  })

  const { data: members } = useQuery({
    queryKey: ['teamMembers', teamId],
    queryFn: () => teamsApi.getMembers(teamId!),
    enabled: !!teamId,
  })

  const { data: teamRotiStats, isLoading: rotiLoading } = useQuery({
    queryKey: ['teamRotiStats', teamId, limit],
    queryFn: () => statsApi.getTeamRotiStats(teamId!, limit),
    enabled: !!teamId && activeTab === 'team',
  })

  const { data: teamMoodStats, isLoading: moodLoading } = useQuery({
    queryKey: ['teamMoodStats', teamId, limit],
    queryFn: () => statsApi.getTeamMoodStats(teamId!, limit),
    enabled: !!teamId && activeTab === 'team',
  })

  const { data: userStats, isLoading: userStatsLoading } = useQuery({
    queryKey: ['userStats', teamId, selectedMemberId, limit],
    queryFn: () => statsApi.getMyStats(teamId!, limit),
    enabled: !!teamId && activeTab === 'individual' && !selectedMemberId,
  })

  const { data: selectedUserRotiStats } = useQuery({
    queryKey: ['userRotiStats', teamId, selectedMemberId, limit],
    queryFn: () => statsApi.getUserRotiStats(teamId!, selectedMemberId!, limit),
    enabled: !!teamId && !!selectedMemberId && activeTab === 'individual',
  })

  const { data: selectedUserMoodStats } = useQuery({
    queryKey: ['userMoodStats', teamId, selectedMemberId, limit],
    queryFn: () => statsApi.getUserMoodStats(teamId!, selectedMemberId!, limit),
    enabled: !!teamId && !!selectedMemberId && activeTab === 'individual',
  })

  const getDominantMood = (distribution: Record<MoodWeather, number> | undefined): MoodWeather | null => {
    if (!distribution) return null
    let maxCount = 0
    let dominant: MoodWeather | null = null
    for (const [mood, count] of Object.entries(distribution)) {
      if (count > maxCount) {
        maxCount = count
        dominant = mood as MoodWeather
      }
    }
    return dominant
  }

  if (teamLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
      </div>
    )
  }

  if (!team) {
    return <div>Équipe non trouvée</div>
  }

  const currentRotiStats = selectedMemberId ? selectedUserRotiStats : userStats?.rotiStats
  const currentMoodStats = selectedMemberId ? selectedUserMoodStats : userStats?.moodStats

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center gap-4">
          <Link
            to={`/teams/${teamId}`}
            className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
          >
            <ArrowLeft className="w-5 h-5 text-gray-600" />
          </Link>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Statistiques - {team.name}</h1>
            <p className="text-gray-500">Analyse des ROTI et humeurs</p>
          </div>
        </div>
      </div>

      {/* Tabs and Filter */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div className="flex bg-gray-100 rounded-lg p-1">
          <button
            onClick={() => setActiveTab('team')}
            className={clsx(
              'flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors',
              activeTab === 'team'
                ? 'bg-white text-gray-900 shadow-sm'
                : 'text-gray-600 hover:text-gray-900'
            )}
          >
            <Users className="w-4 h-4" />
            Équipe
          </button>
          <button
            onClick={() => setActiveTab('individual')}
            className={clsx(
              'flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors',
              activeTab === 'individual'
                ? 'bg-white text-gray-900 shadow-sm'
                : 'text-gray-600 hover:text-gray-900'
            )}
          >
            <User className="w-4 h-4" />
            Individuel
          </button>
        </div>

        <div className="flex items-center gap-4">
          {activeTab === 'individual' && members && (
            <select
              value={selectedMemberId || ''}
              onChange={(e) => setSelectedMemberId(e.target.value || null)}
              className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-transparent"
            >
              <option value="">Mes statistiques</option>
              {members.map((member: TeamMember) => (
                <option key={member.userId} value={member.userId}>
                  {member.user?.displayName || member.userId}
                </option>
              ))}
            </select>
          )}

          <select
            value={periodFilter}
            onChange={(e) => setPeriodFilter(e.target.value as PeriodFilter)}
            className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          >
            {PERIOD_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Team Stats */}
      {activeTab === 'team' && (
        <div className="space-y-6">
          {rotiLoading || moodLoading ? (
            <div className="flex items-center justify-center h-64">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
            </div>
          ) : (
            <>
              {/* Summary Cards */}
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <StatCard
                  title="ROTI moyen"
                  value={teamRotiStats?.average.toFixed(2) || '-'}
                  subtitle={`Sur ${teamRotiStats?.totalRetros || 0} rétro${(teamRotiStats?.totalRetros || 0) > 1 ? 's' : ''}`}
                  icon={<Star className="w-5 h-5" />}
                />
                <StatCard
                  title="Votes ROTI"
                  value={teamRotiStats?.totalVotes || 0}
                  subtitle={`Participation: ${teamRotiStats?.participationRate.toFixed(0) || 0}%`}
                  icon={<TrendingUp className="w-5 h-5" />}
                />
                <StatCard
                  title="Humeur dominante"
                  value={
                    getDominantMood(teamMoodStats?.distribution)
                      ? MOOD_LABELS[getDominantMood(teamMoodStats?.distribution)!]
                      : '-'
                  }
                  subtitle={`${teamMoodStats?.totalMoods || 0} humeurs enregistrées`}
                  icon={<Cloud className="w-5 h-5" />}
                />
                <StatCard
                  title="Participation météo"
                  value={`${teamMoodStats?.participationRate.toFixed(0) || 0}%`}
                  subtitle={`Sur ${teamMoodStats?.totalRetros || 0} rétro${(teamMoodStats?.totalRetros || 0) > 1 ? 's' : ''}`}
                  icon={<Percent className="w-5 h-5" />}
                />
              </div>

              {/* Charts */}
              <div className="grid md:grid-cols-2 gap-6">
                <RotiDistributionChart distribution={teamRotiStats?.distribution || {}} />
                <MoodDistributionChart distribution={teamMoodStats?.distribution || {}} />
              </div>

              <div className="grid md:grid-cols-2 gap-6">
                <RotiEvolutionChart evolution={teamRotiStats?.evolution || []} />
                <MoodEvolutionChart evolution={teamMoodStats?.evolution || []} />
              </div>
            </>
          )}
        </div>
      )}

      {/* Individual Stats */}
      {activeTab === 'individual' && (
        <div className="space-y-6">
          {userStatsLoading ? (
            <div className="flex items-center justify-center h-64">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600" />
            </div>
          ) : (
            <>
              {/* Summary Cards */}
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <StatCard
                  title="Mon ROTI moyen"
                  value={currentRotiStats?.average.toFixed(2) || '-'}
                  subtitle={`Équipe: ${currentRotiStats?.teamAverage?.toFixed(2) || '-'}`}
                  icon={<Star className="w-5 h-5" />}
                  trend={
                    currentRotiStats?.average && currentRotiStats?.teamAverage
                      ? currentRotiStats.average > currentRotiStats.teamAverage
                        ? 'up'
                        : currentRotiStats.average < currentRotiStats.teamAverage
                        ? 'down'
                        : 'neutral'
                      : undefined
                  }
                  trendValue={
                    currentRotiStats?.average && currentRotiStats?.teamAverage
                      ? `${Math.abs(currentRotiStats.average - currentRotiStats.teamAverage).toFixed(2)} vs équipe`
                      : undefined
                  }
                />
                <StatCard
                  title="Rétros participées"
                  value={currentRotiStats?.retrosAttended || 0}
                  subtitle={`${currentRotiStats?.totalVotes || 0} votes ROTI`}
                  icon={<Users className="w-5 h-5" />}
                />
                <StatCard
                  title="Humeur fréquente"
                  value={
                    currentMoodStats?.mostCommonMood
                      ? MOOD_LABELS[currentMoodStats.mostCommonMood]
                      : '-'
                  }
                  subtitle={`${currentMoodStats?.totalMoods || 0} humeurs`}
                  icon={<Cloud className="w-5 h-5" />}
                />
                <StatCard
                  title="Participation"
                  value={`${currentRotiStats?.participationRate?.toFixed(0) || 0}%`}
                  subtitle="Taux de vote ROTI"
                  icon={<Percent className="w-5 h-5" />}
                />
              </div>

              {/* Charts */}
              <div className="grid md:grid-cols-2 gap-6">
                <RotiDistributionChart distribution={currentRotiStats?.distribution || {}} />
                <MoodDistributionChart distribution={currentMoodStats?.distribution || {}} />
              </div>

              <div className="grid md:grid-cols-2 gap-6">
                <RotiEvolutionChart
                  evolution={currentRotiStats?.evolution || []}
                  showAverage={true}
                  averageLabel="Ma moyenne"
                />
                <MoodEvolutionChart evolution={currentMoodStats?.evolution || []} />
              </div>
            </>
          )}
        </div>
      )}
    </div>
  )
}
