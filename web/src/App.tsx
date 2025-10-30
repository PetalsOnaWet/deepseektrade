import { useEffect, useMemo, useState } from 'react';
import useSWR from 'swr';
import { api } from './lib/api';
import { EquityChart } from './components/EquityChart';
import AILearning from './components/AILearning';
import { LanguageProvider, useLanguage } from './contexts/LanguageContext';
import { t, type Language } from './i18n/translations';
import type {
  SystemStatus,
  AccountInfo,
  Position,
  DecisionRecord,
  Statistics,
  TraderInfo,
} from './types';

function App() {
  const { language, setLanguage } = useLanguage();
  useEffect(() => {
    const path = window.location.pathname;
    if (path.startsWith('/zh')) {
      setLanguage('zh');
    } else {
      setLanguage('en');
    }
  }, [setLanguage]);
  const switchLanguage = (lang: Language) => {
    if (lang === language) return;
    if (typeof window !== 'undefined') {
      localStorage.setItem('language', lang);
      window.location.href = lang === 'zh' ? '/zh/' : '/';
    } else {
      setLanguage(lang);
    }
  };
  const [selectedTraderId, setSelectedTraderId] = useState<string | undefined>();
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const [lastUpdate, setLastUpdate] = useState<string>('--:--:--');

  // 获取trader列表
  const { data: traders } = useSWR<TraderInfo[]>('traders', api.getTraders, {
    refreshInterval: 10000,
  });

  // 当获取到traders后，设置默认选中第一个
  useEffect(() => {
    if (traders && traders.length > 0 && !selectedTraderId) {
      setSelectedTraderId(traders[0].trader_id);
    }
  }, [traders, selectedTraderId]);

  // 如果在trader页面，获取该trader的数据
  const { data: status } = useSWR<SystemStatus>(
    selectedTraderId ? `status-${selectedTraderId}` : null,
    () => api.getStatus(selectedTraderId),
    {
      refreshInterval: 5000,
      revalidateOnFocus: true,
      dedupingInterval: 0,
    }
  );

  const { data: account } = useSWR<AccountInfo>(
    selectedTraderId ? `account-${selectedTraderId}` : null,
    () => api.getAccount(selectedTraderId),
    {
      refreshInterval: 5000,
      revalidateOnFocus: true,
      dedupingInterval: 0,
    }
  );

  const { data: positions } = useSWR<Position[]>(
    selectedTraderId ? `positions-${selectedTraderId}` : null,
    () => api.getPositions(selectedTraderId),
    {
      refreshInterval: 5000,
      revalidateOnFocus: true,
      dedupingInterval: 0,
    }
  );

  const { data: decisions } = useSWR<DecisionRecord[]>(
    selectedTraderId ? `decisions-${selectedTraderId}` : null,
    () => api.getDecisions(selectedTraderId),
    { refreshInterval: 15000 }
  );

  const { data: stats } = useSWR<Statistics>(
    selectedTraderId ? `statistics-${selectedTraderId}` : null,
    () => api.getStatistics(selectedTraderId),
    { refreshInterval: 10000 }
  );

  useEffect(() => {
    if (account) {
      const now = new Date().toLocaleTimeString();
      setLastUpdate(now);
    }
  }, [account]);

  const selectedTrader = traders?.find((t) => t.trader_id === selectedTraderId);
  const referralItems = [
    {
      href: 'https://www.maxweb.red/join?ref=BTCB888',
      title: t('referralSpotTitle', language),
      subtitle: t('referralSpotSubtitle', language),
      emoji: '🚀',
    },
    {
      href: 'https://web3.binance.com/referral?ref=BTCB888',
      title: t('referralWalletTitle', language),
      subtitle: t('referralWalletSubtitle', language),
      emoji: '💎',
    },
  ];

  return (
    <div className="min-h-screen" style={{ background: '#0B0E11', color: '#EAECEF' }}>
      {/* Header - Binance Style */}
      <header className="glass sticky top-0 z-50 backdrop-blur-xl">
        <div className="max-w-[1920px] mx-auto px-4 sm:px-6 py-4">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div className="flex items-center justify-between">
              <div className="flex items-start gap-3 sm:items-center">
                <div className="w-8 h-8 rounded-full flex items-center justify-center text-xl shrink-0" style={{ background: 'linear-gradient(135deg, #F0B90B 0%, #FCD535 100%)' }}>
                ⚡
              </div>
              <div>
                <h1 className="text-xl font-bold" style={{ color: '#EAECEF' }}>
                  {t('appTitle', language)}
                </h1>
                <p className="text-xs mono" style={{ color: '#848E9C' }}>
                  {t('subtitle', language)}
                </p>
              </div>
            </div>
              <button
                className="lg:hidden px-3 py-2 rounded text-sm font-semibold"
                style={{ background: '#1E2329', color: '#EAECEF', border: '1px solid #2B3139' }}
                onClick={() => setIsMenuOpen((prev) => !prev)}
              >
                {isMenuOpen ? '✕' : '☰'}
              </button>
            </div>

            <div className="hidden lg:flex flex-col gap-3 w-full lg:w-auto">
              <div className="hidden lg:grid grid-cols-2 gap-2 w-full">
                {referralItems.map((item) => (
                  <a
                    key={item.href}
                    href={item.href}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex flex-col px-3 py-2 rounded-lg transition-all duration-200 hover:-translate-y-0.5"
                    style={{
                      background: 'linear-gradient(135deg, rgba(240, 185, 11, 0.15) 0%, rgba(252, 213, 53, 0.05) 100%)',
                      border: '1px solid rgba(240, 185, 11, 0.35)',
                      boxShadow: '0 6px 18px rgba(240, 185, 11, 0.18)',
                    }}
                  >
                    <span className="text-xs font-semibold flex items-center gap-1" style={{ color: '#F0B90B' }}>
                      <span>{item.emoji}</span>
                      {item.title}
                    </span>
                    <span className="text-[11px] mt-1 leading-snug" style={{ color: '#EAECEF' }}>
                      {item.subtitle}
                    </span>
                  </a>
              ))}
            </div>
            </div>

            {/* Desktop controls */}
            <div className="hidden lg:flex flex-wrap items-center gap-3 justify-between">
              <a
                href="https://github.com/tinkle-community/nofx"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-2 px-3 py-2 rounded text-sm font-semibold transition-all hover:scale-105"
                style={{ background: '#1E2329', color: '#848E9C', border: '1px solid #2B3139' }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = '#2B3139';
                  e.currentTarget.style.color = '#EAECEF';
                  e.currentTarget.style.borderColor = '#F0B90B';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = '#1E2329';
                  e.currentTarget.style.color = '#848E9C';
                  e.currentTarget.style.borderColor = '#2B3139';
                }}
              >
                <svg width="20" height="20" viewBox="0 0 16 16" fill="currentColor">
                  <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
                </svg>
                <span>GitHub</span>
              </a>
              <div className="flex gap-1 rounded p-1" style={{ background: '#1E2329' }}>
                <button
                  onClick={() => switchLanguage('zh')}
                  className="px-3 py-1.5 rounded text-xs font-semibold transition-all"
                  style={language === 'zh'
                    ? { background: '#F0B90B', color: '#000' }
                    : { background: 'transparent', color: '#848E9C' }
                  }
                >
                  中文
                </button>
                <button
                  onClick={() => switchLanguage('en')}
                  className="px-3 py-1.5 rounded text-xs font-semibold transition-all"
                  style={language === 'en'
                    ? { background: '#F0B90B', color: '#000' }
                    : { background: 'transparent', color: '#848E9C' }
                  }
                >
                  EN
                </button>
              </div>

              {status && (
                <div
                  className="flex items-center gap-2 px-3 py-2 rounded justify-center"
                  style={status.is_running
                    ? { background: 'rgba(14, 203, 129, 0.1)', color: '#0ECB81', border: '1px solid rgba(14, 203, 129, 0.2)' }
                    : { background: 'rgba(246, 70, 93, 0.1)', color: '#F6465D', border: '1px solid rgba(246, 70, 93, 0.2)' }
                  }
                >
                  <div
                    className={`w-2 h-2 rounded-full ${status.is_running ? 'pulse-glow' : ''}`}
                    style={{ background: status.is_running ? '#0ECB81' : '#F6465D' }}
                  />
                  <span className="font-semibold mono text-xs">
                    {t(status.is_running ? 'running' : 'stopped', language)}
                  </span>
                </div>
              )}
            </div>

            {/* Mobile menu */}
            {isMenuOpen && (
              <div
                className="lg:hidden mt-3 rounded-xl p-4 flex flex-col gap-3"
                style={{ background: '#1E2329', border: '1px solid #2B3139' }}
              >
                <a
                  href="https://github.com/tinkle-community/nofx"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-2 px-3 py-2 rounded text-sm font-semibold transition-all justify-center"
                  style={{ background: '#0B0E11', color: '#848E9C', border: '1px solid #2B3139' }}
                  onClick={() => setIsMenuOpen(false)}
                >
                  <svg width="20" height="20" viewBox="0 0 16 16" fill="currentColor">
                    <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
                </svg>
                  <span>GitHub</span>
                </a>

                <div className="flex gap-1 rounded p-1" style={{ background: '#0B0E11', border: '1px solid #2B3139' }}>
                  <button
                    onClick={() => switchLanguage('zh')}
                    className="flex-1 px-3 py-1.5 rounded text-xs font-semibold transition-all"
                    style={language === 'zh'
                      ? { background: '#F0B90B', color: '#000' }
                      : { background: 'transparent', color: '#848E9C' }
                    }
                  >
                    中文
                  </button>
                  <button
                    onClick={() => switchLanguage('en')}
                    className="flex-1 px-3 py-1.5 rounded text-xs font-semibold transition-all"
                    style={language === 'en'
                      ? { background: '#F0B90B', color: '#000' }
                      : { background: 'transparent', color: '#848E9C' }
                    }
                  >
                    EN
                  </button>
                </div>

                {status && (
                  <div
                    className="flex items-center gap-2 px-3 py-2 rounded justify-center"
                    style={status.is_running
                      ? { background: 'rgba(14, 203, 129, 0.1)', color: '#0ECB81', border: '1px solid rgba(14, 203, 129, 0.2)' }
                      : { background: 'rgba(246, 70, 93, 0.1)', color: '#F6465D', border: '1px solid rgba(246, 70, 93, 0.2)' }
                    }
                  >
                    <div
                      className={`w-2 h-2 rounded-full ${status.is_running ? 'pulse-glow' : ''}`}
                      style={{ background: status.is_running ? '#0ECB81' : '#F6465D' }}
                    />
                    <span className="font-semibold mono text-xs">
                      {t(status.is_running ? 'running' : 'stopped', language)}
                    </span>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-[1920px] mx-auto px-4 sm:px-6 py-6">
        <TraderDetailsPage
          selectedTrader={selectedTrader}
          status={status}
          account={account}
          positions={positions}
          decisions={decisions}
          stats={stats}
          lastUpdate={lastUpdate}
          language={language}
        />
      </main>

      {/* Footer */}
      <footer className="mt-16" style={{ borderTop: '1px solid #2B3139', background: '#181A20' }}>
        <div className="max-w-[1920px] mx-auto px-6 py-6 text-center text-sm" style={{ color: '#5E6673' }}>
          <div className="grid grid-cols-1 gap-2 mb-4 lg:hidden">
            {referralItems.map((item) => (
              <a
                key={item.href}
                href={item.href}
                target="_blank"
                rel="noopener noreferrer"
                className="flex flex-col px-3 py-2 rounded-lg transition-all duration-200"
                style={{
                  background: 'linear-gradient(135deg, rgba(240, 185, 11, 0.12) 0%, rgba(252, 213, 53, 0.05) 100%)',
                  border: '1px solid rgba(240, 185, 11, 0.25)',
                  boxShadow: '0 4px 14px rgba(240, 185, 11, 0.15)',
                }}
              >
                <span className="text-xs font-semibold flex items-center gap-1 justify-center" style={{ color: '#F0B90B' }}>
                  <span>{item.emoji}</span>
                  {item.title}
                </span>
                <span className="text-[11px] mt-1 leading-snug" style={{ color: '#EAECEF' }}>
                  {item.subtitle}
                </span>
              </a>
            ))}
          </div>
          <p>{t('footerTitle', language)}</p>
          <p className="mt-1">{t('footerWarning', language)}</p>
          <p className="mt-4 text-xs" style={{ color: '#848E9C' }}>
            Powered by adaptive AI trend insights · Monitor · Risk-control · Execute
          </p>
        </div>
      </footer>
    </div>
  );
}

// Trader Details Page Component
function TraderDetailsPage({
  selectedTrader,
  status,
  account,
  positions,
  decisions,
  stats,
  lastUpdate,
  language,
}: {
  selectedTrader?: TraderInfo;
  status?: SystemStatus;
  account?: AccountInfo;
  positions?: Position[];
  decisions?: DecisionRecord[];
  stats?: Statistics;
  lastUpdate: string;
  language: Language;
}) {
  if (!selectedTrader) {
    return (
      <div className="space-y-6">
        {/* Loading Skeleton - Binance Style */}
        <div className="binance-card p-6 animate-pulse">
          <div className="skeleton h-8 w-48 mb-3"></div>
          <div className="flex gap-4">
            <div className="skeleton h-4 w-32"></div>
            <div className="skeleton h-4 w-24"></div>
            <div className="skeleton h-4 w-28"></div>
          </div>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="binance-card p-5 animate-pulse">
              <div className="skeleton h-4 w-24 mb-3"></div>
              <div className="skeleton h-8 w-32"></div>
            </div>
          ))}
        </div>
        <div className="binance-card p-6 animate-pulse">
          <div className="skeleton h-6 w-40 mb-4"></div>
          <div className="skeleton h-64 w-full"></div>
        </div>
      </div>
    );
  }

  const pageSize = 10;
  const [currentDecisionPage, setCurrentDecisionPage] = useState(1);

  useEffect(() => {
    setCurrentDecisionPage(1);
  }, [selectedTrader.trader_id]);

  const sortedDecisions = useMemo<DecisionRecord[]>(() => {
    if (!decisions || decisions.length === 0) {
      return [];
    }
    const cloned = [...decisions];
    cloned.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
    return cloned;
  }, [decisions]);

  useEffect(() => {
    const totalPages = Math.max(1, Math.ceil(sortedDecisions.length / pageSize));
    if (currentDecisionPage > totalPages) {
      setCurrentDecisionPage(totalPages);
    }
  }, [sortedDecisions, currentDecisionPage]);

  const displayedDecisions = useMemo<DecisionRecord[]>(() => {
    if (sortedDecisions.length === 0) {
      return [];
    }
    const start = (currentDecisionPage - 1) * pageSize;
    return sortedDecisions.slice(start, start + pageSize);
  }, [sortedDecisions, currentDecisionPage]);

  const decisionTotalPages = Math.max(1, Math.ceil(sortedDecisions.length / pageSize));

  const baselineBalance = account?.initial_balance ?? status?.initial_balance ?? 20;

  return (
    <div>
      {/* Trader Header */}
      <div className="mb-6 rounded p-6 animate-scale-in" style={{ background: 'linear-gradient(135deg, rgba(240, 185, 11, 0.15) 0%, rgba(252, 213, 53, 0.05) 100%)', border: '1px solid rgba(240, 185, 11, 0.2)', boxShadow: '0 0 30px rgba(240, 185, 11, 0.15)' }}>
        <h2 className="text-2xl font-bold mb-3 flex items-center gap-2" style={{ color: '#EAECEF' }}>
          <span className="w-10 h-10 rounded-full flex items-center justify-center text-xl" style={{ background: 'linear-gradient(135deg, #F0B90B 0%, #FCD535 100%)' }}>
            🤖
          </span>
          {selectedTrader.trader_name}
        </h2>
        <div className="flex items-center gap-4 text-sm" style={{ color: '#848E9C' }}>
          <span>AI Model: <span className="font-semibold" style={{ color: selectedTrader.ai_model === 'qwen' ? '#c084fc' : '#60a5fa' }}>{selectedTrader.ai_model.toUpperCase()}</span></span>
          {status && (
            <>
              <span>•</span>
              <span>Cycles: {status.call_count}</span>
              <span>•</span>
              <span>Runtime: {status.runtime_minutes} min</span>
            </>
          )}
        </div>
      </div>

      {/* Debug Info */}
      {account && (
        <div className="mb-4 p-3 rounded text-xs font-mono" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
          <div style={{ color: '#848E9C' }}>
            🔄 Last Update: {lastUpdate} | Total Equity: {account.total_equity?.toFixed(2) || '0.00'} |
            Available: {account.available_balance?.toFixed(2) || '0.00'} | P&L: {account.total_pnl?.toFixed(2) || '0.00'}{' '}
            ({account.total_pnl_pct?.toFixed(2) || '0.00'}%)
          </div>
        </div>
      )}

      {/* Account Overview */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
        <StatCard
          title={t('totalEquity', language)}
          value={`${account?.total_equity?.toFixed(2) || '0.00'} USDT`}
          change={account?.total_pnl_pct || 0}
          positive={(account?.total_pnl ?? 0) > 0}
          subtitle={`${t('initialBalance', language)}: ${baselineBalance.toFixed(2)} USDT`}
        />
        <StatCard
          title={t('availableBalance', language)}
          value={`${account?.available_balance?.toFixed(2) || '0.00'} USDT`}
          subtitle={`${(account?.available_balance && account?.total_equity ? ((account.available_balance / account.total_equity) * 100).toFixed(1) : '0.0')}% ${t('free', language)}`}
        />
        <StatCard
          title={t('totalPnL', language)}
          value={`${account?.total_pnl !== undefined && account.total_pnl >= 0 ? '+' : ''}${account?.total_pnl?.toFixed(2) || '0.00'} USDT`}
          change={account?.total_pnl_pct || 0}
          positive={(account?.total_pnl ?? 0) >= 0}
        />
        <StatCard
          title={t('positions', language)}
          value={`${account?.position_count || 0}`}
          subtitle={`${t('margin', language)}: ${account?.margin_used_pct?.toFixed(1) || '0.0'}%`}
        />
      </div>

      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
          <StatCard
            title={t('entriesExecuted', language)}
            value={`${stats.total_open_positions}`}
            subtitle={t('sinceLaunch', language)}
          />
          <StatCard
            title={t('exitsExecuted', language)}
            value={`${stats.total_close_positions}`}
            subtitle={t('sinceLaunch', language)}
          />
          <StatCard
            title={t('winRate', language)}
            value={`${stats.win_rate?.toFixed(1) || '0.0'}%`}
            subtitle={t('winsLosses', language, {
              wins: stats.winning_trades ?? 0,
              losses: stats.losing_trades ?? 0,
            })}
          />
        </div>
      )}

      {/* 主要内容区：左右分屏 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        {/* 左侧：图表 + 持仓 */}
        <div className="space-y-6">
          {/* Equity Chart */}
          <div className="animate-slide-in" style={{ animationDelay: '0.1s' }}>
            <EquityChart traderId={selectedTrader.trader_id} />
          </div>

          {/* Current Positions */}
          <div className="binance-card p-6 animate-slide-in" style={{ animationDelay: '0.15s' }}>
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-xl font-bold flex items-center gap-2" style={{ color: '#EAECEF' }}>
            📈 {t('currentPositions', language)}
          </h2>
          {positions && positions.length > 0 && (
            <div className="text-xs px-3 py-1 rounded" style={{ background: 'rgba(240, 185, 11, 0.1)', color: '#F0B90B', border: '1px solid rgba(240, 185, 11, 0.2)' }}>
              {positions.length} {t('active', language)}
            </div>
          )}
        </div>
        {positions && positions.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-left border-b border-gray-800">
                <tr>
                  <th className="pb-3 font-semibold text-gray-400">{t('symbol', language)}</th>
                  <th className="pb-3 font-semibold text-gray-400">{t('side', language)}</th>
                  <th className="pb-3 font-semibold text-gray-400">{t('entryPrice', language)}</th>
                  <th className="pb-3 font-semibold text-gray-400">{t('markPrice', language)}</th>
                  <th className="pb-3 font-semibold text-gray-400">{t('quantity', language)}</th>
                  <th className="pb-3 font-semibold text-gray-400">{t('positionValue', language)}</th>
                  <th className="pb-3 font-semibold text-gray-400">{t('leverage', language)}</th>
                  <th className="pb-3 font-semibold text-gray-400">{t('unrealizedPnL', language)}</th>
                  <th className="pb-3 font-semibold text-gray-400">{t('liqPrice', language)}</th>
                </tr>
              </thead>
              <tbody>
                {positions.map((pos, i) => (
                  <tr key={i} className="border-b border-gray-800 last:border-0">
                    <td className="py-3 font-mono font-semibold">{pos.symbol}</td>
                    <td className="py-3">
                      <span
                        className="px-2 py-1 rounded text-xs font-bold"
                        style={pos.side === 'long'
                          ? { background: 'rgba(14, 203, 129, 0.1)', color: '#0ECB81' }
                          : { background: 'rgba(246, 70, 93, 0.1)', color: '#F6465D' }
                        }
                      >
                        {t(pos.side === 'long' ? 'long' : 'short', language)}
                      </span>
                    </td>
                    <td className="py-3 font-mono" style={{ color: '#EAECEF' }}>{pos.entry_price.toFixed(4)}</td>
                    <td className="py-3 font-mono" style={{ color: '#EAECEF' }}>{pos.mark_price.toFixed(4)}</td>
                    <td className="py-3 font-mono" style={{ color: '#EAECEF' }}>{pos.quantity.toFixed(4)}</td>
                    <td className="py-3 font-mono font-bold" style={{ color: '#EAECEF' }}>
                      {(pos.quantity * pos.mark_price).toFixed(2)} USDT
                    </td>
                    <td className="py-3 font-mono" style={{ color: '#F0B90B' }}>{pos.leverage}x</td>
                    <td className="py-3 font-mono">
                      <span
                        style={{ color: pos.unrealized_pnl >= 0 ? '#0ECB81' : '#F6465D', fontWeight: 'bold' }}
                      >
                        {pos.unrealized_pnl >= 0 ? '+' : ''}
                        {pos.unrealized_pnl.toFixed(2)} ({pos.unrealized_pnl_pct.toFixed(2)}%)
                      </span>
                    </td>
                    <td className="py-3 font-mono" style={{ color: '#848E9C' }}>
                      {pos.liquidation_price.toFixed(4)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="text-center py-16" style={{ color: '#848E9C' }}>
            <div className="text-6xl mb-4 opacity-50">📊</div>
            <div className="text-lg font-semibold mb-2">{t('noPositions', language)}</div>
            <div className="text-sm">{t('noActivePositions', language)}</div>
          </div>
        )}
          </div>
        </div>
        {/* 左侧结束 */}

        {/* 右侧：Recent Decisions - 卡片容器 */}
        <div className="binance-card p-6 animate-slide-in h-fit lg:sticky lg:top-24 lg:max-h-[calc(100vh-120px)]" style={{ animationDelay: '0.2s' }}>
          {/* 标题 */}
          <div className="flex items-center gap-3 mb-5 pb-4 border-b" style={{ borderColor: '#2B3139' }}>
            <div className="w-10 h-10 rounded-xl flex items-center justify-center text-xl" style={{
              background: 'linear-gradient(135deg, #6366F1 0%, #8B5CF6 100%)',
              boxShadow: '0 4px 14px rgba(99, 102, 241, 0.4)'
            }}>
              🧠
            </div>
            <div>
              <h2 className="text-xl font-bold" style={{ color: '#EAECEF' }}>{t('recentDecisions', language)}</h2>
              {sortedDecisions.length > 0 && (
                <div className="text-xs" style={{ color: '#848E9C' }}>
                  {t('lastCycles', language, { count: sortedDecisions.length })}
                </div>
              )}
            </div>
          </div>

          {/* 决策列表 - 可滚动 */}
          <div className="space-y-4 overflow-y-auto pr-2" style={{ maxHeight: 'calc(100vh - 280px)' }}>
            {displayedDecisions.length > 0 ? (
              displayedDecisions.map((decision, i) => (
                <DecisionCard key={i} decision={decision} language={language} />
              ))
            ) : (
              <div className="py-16 text-center">
                <div className="text-6xl mb-4 opacity-30">🧠</div>
                <div className="text-lg font-semibold mb-2" style={{ color: '#EAECEF' }}>{t('noDecisionsYet', language)}</div>
                <div className="text-sm" style={{ color: '#848E9C' }}>{t('aiDecisionsWillAppear', language)}</div>
              </div>
            )}
          </div>
          {sortedDecisions.length > 0 && (
            <div className="flex items-center justify-between mt-4 px-2 text-xs" style={{ color: '#848E9C' }}>
              <button
                className="px-3 py-1 rounded border transition"
                style={
                  currentDecisionPage === 1
                    ? { borderColor: '#2B3139', color: '#2B3139', cursor: 'not-allowed' }
                    : { borderColor: '#2B3139', color: '#EAECEF' }
                }
                disabled={currentDecisionPage === 1}
                onClick={() => setCurrentDecisionPage((prev) => Math.max(1, prev - 1))}
              >
                {t('prevPage', language)}
              </button>
              <div className="font-mono">
                {t('decisionsPage', language, { current: currentDecisionPage, total: decisionTotalPages })}
              </div>
              <button
                className="px-3 py-1 rounded border transition"
                style={
                  currentDecisionPage >= decisionTotalPages
                    ? { borderColor: '#2B3139', color: '#2B3139', cursor: 'not-allowed' }
                    : { borderColor: '#2B3139', color: '#EAECEF' }
                }
                disabled={currentDecisionPage >= decisionTotalPages}
                onClick={() =>
                  setCurrentDecisionPage((prev) =>
                    Math.min(decisionTotalPages, prev + 1)
                  )
                }
              >
                {t('nextPage', language)}
              </button>
            </div>
          )}
        </div>
        {/* 右侧结束 */}
      </div>

      {/* AI Learning & Performance Analysis */}
      <div className="mb-6 animate-slide-in" style={{ animationDelay: '0.3s' }}>
        <AILearning traderId={selectedTrader.trader_id} />
      </div>

      <SeoSection language={language} />
    </div>
  );
}

// Stat Card Component - Binance Style Enhanced
function StatCard({
  title,
  value,
  change,
  positive,
  subtitle,
}: {
  title: string;
  value: string;
  change?: number;
  positive?: boolean;
  subtitle?: string;
}) {
  return (
    <div className="stat-card animate-fade-in">
      <div className="text-xs mb-2 mono uppercase tracking-wider" style={{ color: '#848E9C' }}>{title}</div>
      <div className="text-2xl font-bold mb-1 mono" style={{ color: '#EAECEF' }}>{value}</div>
      {change !== undefined && (
        <div className="flex items-center gap-1">
          <div
            className="text-sm mono font-bold"
            style={{ color: positive ? '#0ECB81' : '#F6465D' }}
          >
            {positive ? '▲' : '▼'} {positive ? '+' : ''}
            {change.toFixed(2)}%
          </div>
        </div>
      )}
      {subtitle && <div className="text-xs mt-2 mono" style={{ color: '#848E9C' }}>{subtitle}</div>}
    </div>
  );
}

// Decision Card Component with CoT Trace - Binance Style
function DecisionCard({ decision, language }: { decision: DecisionRecord; language: Language }) {
  const [showInputPrompt, setShowInputPrompt] = useState(false);
  const [showCoT, setShowCoT] = useState(false);

  return (
    <div className="rounded p-5 transition-all duration-300 hover:translate-y-[-2px]" style={{ border: '1px solid #2B3139', background: '#1E2329', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.3)' }}>
      {/* Header */}
      <div className="flex items-start justify-between mb-3">
        <div>
          <div className="font-semibold" style={{ color: '#EAECEF' }}>{t('cycle', language)} #{decision.cycle_number}</div>
          <div className="text-xs" style={{ color: '#848E9C' }}>
            {new Date(decision.timestamp).toLocaleString()}
          </div>
        </div>
        <div
          className="px-3 py-1 rounded text-xs font-bold"
          style={decision.success
            ? { background: 'rgba(14, 203, 129, 0.1)', color: '#0ECB81' }
            : { background: 'rgba(246, 70, 93, 0.1)', color: '#F6465D' }
          }
        >
          {t(decision.success ? 'success' : 'failed', language)}
        </div>
      </div>

      {/* Input Prompt - Collapsible */}
      {decision.input_prompt && (
        <div className="mb-3">
          <button
            onClick={() => setShowInputPrompt(!showInputPrompt)}
            className="flex items-center gap-2 text-sm transition-colors"
            style={{ color: '#60a5fa' }}
          >
            <span className="font-semibold">📥 {t('inputPrompt', language)}</span>
            <span className="text-xs">{showInputPrompt ? t('collapse', language) : t('expand', language)}</span>
          </button>
          {showInputPrompt && (
            <div className="mt-2 rounded p-4 text-sm font-mono whitespace-pre-wrap max-h-96 overflow-y-auto" style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}>
              {decision.input_prompt}
            </div>
          )}
        </div>
      )}

      {/* AI Chain of Thought - Collapsible */}
      {decision.cot_trace && (
        <div className="mb-3">
          <button
            onClick={() => setShowCoT(!showCoT)}
            className="flex items-center gap-2 text-sm transition-colors"
            style={{ color: '#F0B90B' }}
          >
            <span className="font-semibold">📤 {t('aiThinking', language)}</span>
            <span className="text-xs">{showCoT ? t('collapse', language) : t('expand', language)}</span>
          </button>
          {showCoT && (
            <div className="mt-2 rounded p-4 text-sm font-mono whitespace-pre-wrap max-h-96 overflow-y-auto" style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}>
              {decision.cot_trace}
            </div>
          )}
        </div>
      )}

      {/* Decisions Actions */}
      {decision.decisions && decision.decisions.length > 0 && (
        <div className="space-y-2 mb-3">
          {decision.decisions.map((action, j) => (
            <div key={j} className="flex items-center gap-2 text-sm rounded px-3 py-2" style={{ background: '#0B0E11' }}>
              <span className="font-mono font-bold" style={{ color: '#EAECEF' }}>{action.symbol}</span>
              <span
                className="px-2 py-0.5 rounded text-xs font-bold"
                style={action.action.includes('open')
                  ? { background: 'rgba(96, 165, 250, 0.1)', color: '#60a5fa' }
                  : { background: 'rgba(240, 185, 11, 0.1)', color: '#F0B90B' }
                }
              >
                {action.action}
              </span>
              {action.leverage > 0 && <span style={{ color: '#F0B90B' }}>{action.leverage}x</span>}
              {action.price > 0 && (
                <span className="font-mono text-xs" style={{ color: '#848E9C' }}>@{action.price.toFixed(4)}</span>
              )}
              <span style={{ color: action.success ? '#0ECB81' : '#F6465D' }}>
                {action.success ? '✓' : '✗'}
              </span>
              {action.error && <span className="text-xs ml-2" style={{ color: '#F6465D' }}>{action.error}</span>}
            </div>
          ))}
        </div>
      )}

      {/* Account State Summary */}
      {decision.account_state && (
        <div className="flex gap-4 text-xs mb-3 rounded px-3 py-2" style={{ background: '#0B0E11', color: '#848E9C' }}>
          <span>净值: {decision.account_state.total_balance.toFixed(2)} USDT</span>
          <span>可用: {decision.account_state.available_balance.toFixed(2)} USDT</span>
          <span>保证金率: {decision.account_state.margin_used_pct.toFixed(1)}%</span>
          <span>持仓: {decision.account_state.position_count}</span>
        </div>
      )}

      {/* Execution Logs */}
      {decision.execution_log && decision.execution_log.length > 0 && (
        <div className="space-y-1">
          {decision.execution_log.map((log, k) => (
            <div
              key={k}
              className="text-xs font-mono"
              style={{ color: log.includes('✓') || log.includes('成功') ? '#0ECB81' : '#F6465D' }}
            >
              {log}
            </div>
          ))}
        </div>
      )}

      {/* Error Message */}
      {decision.error_message && (
        <div className="text-sm rounded px-3 py-2 mt-3" style={{ color: '#F6465D', background: 'rgba(246, 70, 93, 0.1)' }}>
          ❌ {decision.error_message}
        </div>
      )}
    </div>
  );
}

function SeoSection({ language }: { language: Language }) {
  const isZh = language === 'zh';
  const sectionStyle: React.CSSProperties = {
    background: '#0B0E11',
    borderTop: '1px solid #1F262F',
    marginTop: '2.5rem',
  };
  const containerStyle: React.CSSProperties = {
    maxWidth: '1920px',
    margin: '0 auto',
    padding: '2.5rem clamp(1.5rem, 4vw, 4rem)',
    lineHeight: 1.7,
    color: '#EAECEF',
  };
  const headingStyle: React.CSSProperties = {
    marginTop: '1.4rem',
    fontSize: '1.25rem',
    color: '#F8FAFC',
  };
  const listStyle: React.CSSProperties = {
    paddingLeft: '1.2rem',
    marginTop: '0.7rem',
  };
  const taglineStyle: React.CSSProperties = {
    display: 'inline-block',
    padding: '0.25rem 0.75rem',
    borderRadius: '999px',
    background: 'rgba(14, 203, 129, 0.15)',
    color: '#22d3a3',
    fontWeight: 600,
    fontSize: '0.8rem',
    marginBottom: '0.9rem',
  };

  const content = isZh
    ? [
        {
          title: '关于加密货币 AI 交易直播',
          body: [
            '本系统实时展示一个币安 USDT 永续合约账户（当前权益 20 USDT）。所有决策由 DeepSeek 推理模型全权负责。AI 每 3 分钟收集行情、仓位和风险数据，输出结构化交易指令，由执行引擎下单并记录日志，盈亏与策略反思都会第一时间呈现。',
          ],
        },
        {
          title: '系统能做什么？',
          list: [
            '直连币安合约账户，实时拉取余额、仓位与实际盈亏。',
            '展示 AI 的思维链、保护计划与每笔执行日志。',
            '通过 ShareThis 快速生成社交媒体分享链接。',
            '多周期筛选（3 分钟 / 4 小时 / 日线 / 周线）识别趋势机会。',
          ],
        },
        {
          title: '是真实实盘吗？',
          body: [
            '完全是实盘。API Key 指向真实的币安永续子账户，所有仓位、保证金和已实现盈亏均来自币安官方接口，没有模拟盘或历史回放。',
          ],
        },
        {
          title: '使用什么模型？',
          body: [
            '交易策略由 DeepSeek 推理模型驱动。系统对其输出做严格校验：杠杆上限 10×，单笔风险必须低于净值的 3%，并要求提供完整的风控计划，否则指令会被拒绝执行。',
          ],
        },
        {
          title: '核心策略',
          list: [
            '趋势跟随：多周期共振才开仓，弱趋势时宁可观望。',
            '两段式风控：行情获利 3%（未放大杠杆）后，将止损抬到保本位置。',
            '动态追踪：再扩张 1.5%~2% 后，启用 0.8%~1.5% 的波动率自适应追踪止损，并随盈利阶梯收紧。',
            '风险敞口限制：最多持有 3 个品种，总风险敞口不超过净值的 3%。',
          ],
        },
      ]
    : [
        {
          title: 'About Crypto AI Trading Live',
          body: [
            'Crypto AI Trading Live streams a live Binance USDT-margined futures account (current equity 20 USDT). The execution loop is fully autonomous and powered by the DeepSeek reasoning model. Every three minutes the AI evaluates market data, open risk, and positions, then emits structured trade instructions. Executions, P&L, and reflection logs are published here in real time.',
          ],
        },
        {
          title: 'What the system does',
          list: [
            'Connects directly to Binance perpetual futures via API.',
            'Streams live equity, margin usage, and open positions.',
            'Visualises AI chain-of-thought, protection plans, and every execution record.',
            'Generates share-ready social cards through ShareThis.',
          ],
        },
        {
          title: 'Is it real trading?',
          body: [
            'Yes. Orders are signed with a live Binance futures API key. Balance, realised P&L, and position data are fetched from Binance every cycle. There is no paper trading or replayed feed.',
          ],
        },
        {
          title: 'Which model is driving the trades?',
          body: [
            'Decisions come from the DeepSeek reasoning model. Outputs are validated locally: leverage is capped at 10×, per-trade risk stays under 3% of equity, and each JSON response must include a complete protection plan.',
          ],
        },
        {
          title: 'Core strategy',
          list: [
            'Trend-following bias guided by 3 minute, 4 hour, daily, and weekly structure.',
            'Two-stage risk control: once price advances 3% in favour, the stop moves to breakeven (with a small buffer).',
            'Volatility-aware trailing stops kick in after another 1.5–2% move, tightening from 1.5% toward 0.8% as profits grow.',
            'Risk capping: at most three concurrent symbols with exposure limited to 3% of equity per trade.',
          ],
        },
      ];

  return (
    <section className="-mx-4 sm:-mx-6 lg:-mx-10" style={sectionStyle}>
      <div style={containerStyle}>
        <span style={taglineStyle}>{isZh ? '加密货币 AI 实盘直播' : 'Live crypto AI trading dashboard'}</span>
        {content.map((section) => (
          <div key={section.title}>
            <h2 style={headingStyle}>{section.title}</h2>
            {section.body?.map((paragraph, idx) => (
              <p key={idx}>{paragraph}</p>
            ))}
            {section.list && (
              <ul style={listStyle}>
                {section.list.map((item, idx) => (
                  <li key={idx}>{item}</li>
                ))}
              </ul>
            )}
          </div>
        ))}
      </div>
    </section>
  );
}

// Wrap App with LanguageProvider
export default function AppWithLanguage() {
  return (
    <LanguageProvider>
      <App />
    </LanguageProvider>
  );
}
