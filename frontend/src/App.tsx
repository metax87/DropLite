import { useState } from "react";

// 占位的上传队列用于在后端 API 尚未完成时保持界面交互性。
export default function App() {
  const [files, setFiles] = useState<File[]>([]);

  function handleSelected(event: React.ChangeEvent<HTMLInputElement>) {
    if (!event.target.files) return;
    setFiles(Array.from(event.target.files));
  }

  return (
    <main className="app-shell">
      <section>
        <h1>DropLite</h1>
        <p>轻量文件上传服务，前后端开发正处于起步阶段。</p>
      </section>

      <section>
        <label htmlFor="uploader">选择文件：</label>
        <input id="uploader" type="file" multiple onChange={handleSelected} />
      </section>

      <section>
        <h2>待上传列表</h2>
        {files.length === 0 ? (
          <p>尚未选择文件。</p>
        ) : (
          <ul>
            {files.map((file) => (
              <li key={file.name}>
                {file.name} — {(file.size / 1024).toFixed(2)} KB
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  );
}
