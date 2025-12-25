export function ContactPage() {
  return (
    <div className="py-8 max-w-3xl mx-auto">
      <h1 className="page-title">お問い合わせ</h1>

      <div className="space-y-6">
        <section className="card">
          <div className="card-body space-y-2 text-sm leading-relaxed">
            <h2 className="text-lg font-semibold">本システムについて</h2>
            <p className="text-muted">
              本システムは東京情報大学ソフトウェアコンテスト向けに開発されたオンラインジャッジです。
            </p>
          </div>
        </section>

        <section className="card">
          <div className="card-body space-y-2 text-sm leading-relaxed">
            <h2 className="text-lg font-semibold">連絡先</h2>
            <div className="space-y-1">
              <p><strong>作者:</strong> 竹下暖人</p>
              <p><strong>所属:</strong> 東京情報大学</p>
              <p>
                <strong>Email:</strong>
                <span className="ml-2">tuisojcontact あっとまーく gmail.com</span>
              </p>
            </div>
            <p className="text-xs text-muted mt-2">
              ※ スパム対策のため、メールアドレスは「あっとまーく」表記です。「@」に置き換えてご連絡ください。
            </p>
          </div>
        </section>

        <section className="card">
          <div className="card-body space-y-2 text-sm leading-relaxed">
            <h2 className="text-lg font-semibold">お問い合わせ可能な内容</h2>
            <ul className="list-disc list-inside space-y-1">
              <li>バグ報告</li>
              <li>問題文・テストケースの誤り指摘</li>
              <li>システムの使い方に関する質問</li>
            </ul>
          </div>
        </section>
      </div>
    </div>
  )
}
