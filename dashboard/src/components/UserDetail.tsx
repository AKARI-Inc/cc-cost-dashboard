import type { UsageRow } from '../hooks/useUsageData';
import { useUserDetail } from '../hooks/useUserDetail';

type Props = { row: UsageRow; from: string; to: string };

export function UserDetail({ row, from, to }: Props) {
  const email = row.user_email ?? row.key ?? '(unknown)';
  const { models, tools, skills, sessions, loading, error } = useUserDetail({
    userEmail: email,
    from,
    to,
  });

  const inputWithCache = row.input_tokens + row.cache_read_tokens;
  const cacheHitRate = inputWithCache > 0
    ? (row.cache_read_tokens / inputWithCache) * 100
    : 0;
  const ioRatio = row.input_tokens > 0
    ? row.output_tokens / row.input_tokens
    : 0;

  const totalToolCalls = tools.reduce((sum, t) => sum + t.request_count, 0);
  const totalSkillCalls = skills.reduce((sum, s) => sum + s.use_count, 0);

  return (
    <div className="user-detail">
      <section className="detail-section">
        <h4>トークン詳細</h4>
        <dl className="detail-grid">
          <div>
            <dt>キャッシュ読込</dt>
            <dd>{row.cache_read_tokens.toLocaleString()}</dd>
          </div>
          <div>
            <dt>キャッシュ作成</dt>
            <dd>{row.cache_creation_tokens.toLocaleString()}</dd>
          </div>
          <div>
            <dt>キャッシュヒット率</dt>
            <dd>{cacheHitRate.toFixed(1)}%</dd>
          </div>
          <div>
            <dt>入出力比率 (出/入)</dt>
            <dd>{ioRatio.toFixed(2)}</dd>
          </div>
          <div>
            <dt>平均トークン/リクエスト</dt>
            <dd>
              {row.request_count > 0
                ? Math.round(
                    (row.input_tokens + row.output_tokens) / row.request_count,
                  ).toLocaleString()
                : '-'}
            </dd>
          </div>
        </dl>
      </section>

      <section className="detail-section">
        <h4>セッション統計</h4>
        <dl className="detail-grid">
          <div>
            <dt>ユニークセッション数</dt>
            <dd>{sessions.session_count.toLocaleString()}</dd>
          </div>
          <div>
            <dt>1セッション平均リクエスト</dt>
            <dd>
              {sessions.session_count > 0
                ? sessions.avg_requests.toFixed(1)
                : '-'}
            </dd>
          </div>
          <div>
            <dt>1セッション平均コスト</dt>
            <dd>
              {sessions.session_count > 0
                ? `$${sessions.avg_cost_usd.toFixed(4)}`
                : '-'}
            </dd>
          </div>
          <div>
            <dt>最大コスト/セッション</dt>
            <dd>
              {sessions.session_count > 0
                ? `$${sessions.max_cost_usd.toFixed(4)}`
                : '-'}
            </dd>
          </div>
        </dl>
      </section>

      {loading && <p className="info">詳細データ読み込み中...</p>}
      {error && <p className="error">詳細データ取得失敗: {error}</p>}

      {!loading && !error && (
        <>
          <section className="detail-section">
            <h4>モデル別内訳</h4>
            {models.length === 0 ? (
              <p className="info">データなし</p>
            ) : (
              <table className="detail-table">
                <thead>
                  <tr>
                    <th>モデル</th>
                    <th className="num">リクエスト</th>
                    <th className="num">入力</th>
                    <th className="num">出力</th>
                    <th className="num">キャッシュ読</th>
                    <th className="num">コスト</th>
                    <th className="num">コスト比</th>
                  </tr>
                </thead>
                <tbody>
                  {models.map((m) => {
                    const share = row.total_cost_usd > 0
                      ? (m.total_cost_usd / row.total_cost_usd) * 100
                      : 0;
                    return (
                      <tr key={m.model}>
                        <td>{m.model}</td>
                        <td className="num">{m.request_count.toLocaleString()}</td>
                        <td className="num">{m.input_tokens.toLocaleString()}</td>
                        <td className="num">{m.output_tokens.toLocaleString()}</td>
                        <td className="num">{m.cache_read_tokens.toLocaleString()}</td>
                        <td className="num">${m.total_cost_usd.toFixed(4)}</td>
                        <td className="num">{share.toFixed(1)}%</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            )}
          </section>

          <section className="detail-section">
            <h4>
              Skill 利用{' '}
              {skills.length > 0 && (
                <span className="muted">
                  ({skills.length.toLocaleString()} 種 / 全 {totalSkillCalls.toLocaleString()} 回)
                </span>
              )}
            </h4>
            {skills.length === 0 ? (
              <p className="info">Skill 起動データなし</p>
            ) : (
              <div className="tool-scroll">
                <table className="detail-table tool-table">
                  <thead>
                    <tr>
                      <th>Skill</th>
                      <th>ソース</th>
                      <th className="num">回数</th>
                      <th className="num">割合</th>
                      <th className="bar-col">分布</th>
                    </tr>
                  </thead>
                  <tbody>
                    {skills.map((s) => {
                      const pct = totalSkillCalls > 0
                        ? (s.use_count / totalSkillCalls) * 100
                        : 0;
                      const src = s.plugin_name
                        ? `${s.skill_source ?? '-'} / ${s.plugin_name}`
                        : s.skill_source ?? '-';
                      return (
                        <tr key={s.skill_name}>
                          <td>{s.skill_name}</td>
                          <td className="muted">{src}</td>
                          <td className="num">{s.use_count.toLocaleString()}</td>
                          <td className="num">{pct.toFixed(1)}%</td>
                          <td className="bar-col">
                            <div className="bar-track">
                              <div className="bar-fill" style={{ width: `${pct}%` }} />
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <section className="detail-section">
            <h4>
              ツール利用{' '}
              {tools.length > 0 && (
                <span className="muted">
                  ({tools.length.toLocaleString()} 種 / 全 {totalToolCalls.toLocaleString()} 回)
                </span>
              )}
            </h4>
            {tools.length === 0 ? (
              <p className="info">ツール利用データなし</p>
            ) : (
              <div className="tool-scroll">
                <table className="detail-table tool-table">
                  <thead>
                    <tr>
                      <th>ツール</th>
                      <th className="num">回数</th>
                      <th className="num">割合</th>
                      <th className="bar-col">分布</th>
                    </tr>
                  </thead>
                  <tbody>
                    {tools.map((t) => {
                      const pct = totalToolCalls > 0
                        ? (t.request_count / totalToolCalls) * 100
                        : 0;
                      return (
                        <tr key={t.tool_name}>
                          <td>{t.tool_name}</td>
                          <td className="num">{t.request_count.toLocaleString()}</td>
                          <td className="num">{pct.toFixed(1)}%</td>
                          <td className="bar-col">
                            <div className="bar-track">
                              <div className="bar-fill" style={{ width: `${pct}%` }} />
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        </>
      )}
    </div>
  );
}
