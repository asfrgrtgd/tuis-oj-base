import { VerdictBadge } from '@/components/common'

export function HelpPage() {
  return (
    <div className="py-8 max-w-4xl mx-auto">
      <h1 className="page-title">ヘルプ / FAQ</h1>

      <div className="space-y-6">
        {/* はじめての方へ */}
        <section className="card">
          <div className="card-header">
            <h2 className="text-lg font-semibold">はじめての方へ</h2>
          </div>
          <div className="card-body space-y-4 text-sm leading-relaxed">
            <p>
              <strong>TUIS OJ</strong>へようこそ！このサイトはプログラミングの問題を解いて、自動で正誤判定を受けられる<strong>オンラインジャッジシステム</strong>です。
            </p>
            <div>
              <h3 className="font-semibold mb-2">オンラインジャッジとは</h3>
              <p className="text-muted">
                プログラムを提出すると、あらかじめ用意されたテストケースで自動実行され、正解かどうかを判定してくれるシステムです。
                競技プログラミングやプログラミング学習に広く使われています。
              </p>
            </div>
            <div>
              <h3 className="font-semibold mb-2">はじめてみよう</h3>
              <ol className="list-decimal list-inside space-y-1 text-muted">
                <li>問題一覧から挑戦したい問題を選ぶ</li>
                <li>問題文を読んで、解答コードを書く</li>
                <li>「提出」ボタンでコードを送信</li>
                <li>結果を確認（ <VerdictBadge verdict="AC" /> なら正解！）</li>
              </ol>
            </div>
          </div>
        </section>

        {/* 使い方ガイド */}
        <section className="card">
          <div className="card-header">
            <h2 className="text-lg font-semibold">使い方ガイド</h2>
          </div>
          <div className="card-body space-y-4 text-sm leading-relaxed">
            <div>
              <h3 className="font-semibold mb-2">問題を選ぶ</h3>
              <ul className="list-disc list-inside space-y-1 text-muted">
                <li>問題一覧ページから好きな問題を選びましょう</li>
                <li>緑のチェックマーク = 正解済み</li>
                <li>白い丸 = 未挑戦</li>
              </ul>
            </div>
            <div>
              <h3 className="font-semibold mb-2">コードを書く</h3>
              <ul className="list-disc list-inside space-y-1 text-muted">
                <li>問題ページ下部のエディタにコードを入力</li>
                <li>言語は右上のドロップダウンで選択</li>
                <li>標準入力から読み取り、標準出力に出力してください</li>
              </ul>
            </div>
            <div>
              <h3 className="font-semibold mb-2">提出する</h3>
              <ul className="list-disc list-inside space-y-1 text-muted">
                <li>「提出」ボタンをクリック</li>
                <li>数秒〜数十秒で結果が表示されます</li>
              </ul>
            </div>
            <div>
              <h3 className="font-semibold mb-2">結果を確認する</h3>
              <p className="text-muted">
                <VerdictBadge verdict="AC" /> なら正解です！おめでとうございます。
                その他の結果については「ジャッジシステム」を参照してください。
              </p>
            </div>
          </div>
        </section>

        {/* よくある質問 */}
        <section className="card">
          <div className="card-header">
            <h2 className="text-lg font-semibold">よくある質問</h2>
          </div>
          <div className="card-body space-y-4 text-sm leading-relaxed">
            <div>
              <h3 className="font-semibold mb-1">Q. ログインはどうすればいいですか？</h3>
              <p className="text-muted">A. 管理者から配布されたユーザーIDとパスワードでログインしてください。</p>
            </div>
            <div>
              <h3 className="font-semibold mb-1">Q. 提出がPendingのまま動きません</h3>
              <p className="text-muted">
                A. ジャッジが混雑している場合があります。数分待っても変わらない場合は、ページを再読み込みしてください。
                長時間続く場合はお問い合わせください。
              </p>
            </div>
            <div>
              <h3 className="font-semibold mb-1">Q. WA（不正解）になりました。どうすればいいですか？</h3>
              <p className="text-muted">A. 以下を確認してください：</p>
              <ul className="list-disc list-inside space-y-1 text-muted ml-4">
                <li>サンプル入力で正しく動作するか</li>
                <li>出力の末尾に余分な空白や改行がないか</li>
                <li>入出力フォーマットが問題文通りか</li>
              </ul>
            </div>
            <div>
              <h3 className="font-semibold mb-1">Q. RE（実行時エラー）が出ます</h3>
              <p className="text-muted">
                A. 配列の範囲外アクセス、ゼロ除算、スタックオーバーフローなどが原因として考えられます。
              </p>
            </div>
            <div>
              <h3 className="font-semibold mb-1">Q. パスワードを忘れました</h3>
              <p className="text-muted">A. お問い合わせページから管理者にご連絡ください。</p>
            </div>
          </div>
        </section>

        {/* ジャッジシステム */}
        <section className="card">
          <div className="card-header">
            <h2 className="text-lg font-semibold">ジャッジシステム</h2>
          </div>
          <div className="card-body space-y-4 text-sm leading-relaxed">
            <div>
              <h3 className="font-semibold mb-3">判定結果の意味</h3>
              <div className="space-y-3">
                <div className="flex items-start gap-3">
                  <VerdictBadge verdict="AC" />
                  <div>
                    <p><strong>Accepted</strong> - 正解です！すべてのテストケースをパスしました。</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <VerdictBadge verdict="WA" />
                  <div>
                    <p><strong>Wrong Answer</strong> - 出力が期待値と異なります。</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <VerdictBadge verdict="TLE" />
                  <div>
                    <p><strong>Time Limit Exceeded</strong> - 実行時間が制限を超えました。</p>
                    <p className="text-xs text-muted mt-1">
                      ※ コンパイル言語（C++、Java等）の場合、コンパイル時間は含まれません。計測されるのは実行時間のみです。
                    </p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <VerdictBadge verdict="MLE" />
                  <div>
                    <p><strong>Memory Limit Exceeded</strong> - メモリ使用量が制限を超えました。</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <VerdictBadge verdict="RE" />
                  <div>
                    <p><strong>Runtime Error</strong> - 実行時にエラーが発生しました。</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <VerdictBadge verdict="CE" />
                  <div>
                    <p><strong>Compile Error</strong> - コンパイルに失敗しました。</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <VerdictBadge verdict="SE" />
                  <div>
                    <p><strong>System Error</strong> - システム側のエラーです。再提出してください。</p>
                  </div>
                </div>
              </div>
            </div>
            <div>
              <h3 className="font-semibold mb-2">判定の流れ</h3>
              <ol className="list-decimal list-inside space-y-1 text-muted">
                <li>コードを受け取り、コンパイル（該当言語のみ）</li>
                <li>複数のテストケースで順番に実行</li>
                <li>すべてのテストケースで正しい出力が得られれば <VerdictBadge verdict="AC" /></li>
              </ol>
            </div>
          </div>
        </section>

        {/* 対応言語とバージョン */}
        <section className="card">
          <div className="card-header">
            <h2 className="text-lg font-semibold">対応言語とバージョン</h2>
          </div>
          <div className="card-body space-y-3 text-sm leading-relaxed">
            <div className="overflow-x-auto">
              <table className="table text-sm">
                <thead>
                  <tr>
                    <th>言語</th>
                    <th>バージョン</th>
                    <th>コンパイル/実行</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>C++</td>
                    <td className="mono">GCC 12 (GNU++17)</td>
                    <td className="mono text-xs">g++ -std=gnu++17 -O2</td>
                  </tr>
                  <tr>
                    <td>Python</td>
                    <td className="mono">3.11</td>
                    <td className="mono text-xs">python3</td>
                  </tr>
                  <tr>
                    <td>Java</td>
                    <td className="mono">OpenJDK 21</td>
                    <td className="mono text-xs">javac / java</td>
                  </tr>
                  <tr>
                    <td>C</td>
                    <td className="mono">GCC 12 (GNU17)</td>
                    <td className="mono text-xs">gcc -std=gnu17 -O2</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </section>
      </div>
    </div>
  )
}
