import { useMemo, useState, type ChangeEvent } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AuthProvider, useAuth } from "./components/AuthProvider";
import LoginPage from "./pages/LoginPage";
import { supabase } from "./lib/supabase";

const API_BASE = (import.meta.env.VITE_API_BASE ?? "").replace(/\/$/, "");
const FILES_ENDPOINT = `${API_BASE}/files`;

interface FileRecord {
  id: string;
  original_name: string;
  mime_type: string;
  size_bytes: number;
  storage_path: string;
  status: string;
  created_at: string;
}

interface UploadTask {
  id: string;
  filename: string;
  progress: number;
  status: "uploading" | "success" | "error";
  error?: string;
}

function formatBytes(size: number) {
  if (!Number.isFinite(size) || size <= 0) return "-";
  const units = ["B", "KB", "MB", "GB"];
  const exponent = Math.min(Math.floor(Math.log(size) / Math.log(1024)), units.length - 1);
  const value = size / Math.pow(1024, exponent);
  return `${value.toFixed(1)} ${units[exponent]}`;
}

// 主应用界面（仅在登录后渲染）
function MainApp() {
  const { session, user } = useAuth();
  const queryClient = useQueryClient();
  const [uploads, setUploads] = useState<UploadTask[]>([]);
  const [deletingIds, setDeletingIds] = useState<Set<string>>(new Set());

  // 这里的 session 理论上不应该为空，因为 MainApp 由 AuthWrapper 保护
  // 但为了类型安全可以做个简单的断言或 return null（不影响 hooks 顺序）
  if (!session) return null;

  // 获取带 Token 的 Headers
  const getAuthHeaders = () => ({
    Accept: "application/json",
    Authorization: `Bearer ${session.access_token}`
  });

  const fetchFiles = async (): Promise<FileRecord[]> => {
    const response = await fetch(FILES_ENDPOINT, {
      headers: getAuthHeaders()
    });
    if (!response.ok) {
      throw new Error("无法获取文件列表");
    }
    const payload = await response.json();
    return payload?.data ?? [];
  };

  const { data: files = [], isLoading, isFetching, error } = useQuery({
    queryKey: ["files"],
    queryFn: fetchFiles
  });

  const sortedFiles = useMemo(() => {
    return [...files].sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at));
  }, [files]);

  const startUpload = async (file: File) => {
    const uploadId = crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random()}`;
    setUploads((current) => [
      ...current,
      { id: uploadId, filename: file.name, progress: 0, status: "uploading" }
    ]);

    try {
      await new Promise<FileRecord>((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open("POST", FILES_ENDPOINT);
        xhr.setRequestHeader("Authorization", `Bearer ${session.access_token}`);

        xhr.upload.onprogress = (event) => {
          if (!event.lengthComputable) return;
          const percent = Math.round((event.loaded / event.total) * 100);
          setUploads((current) =>
            current.map((task) =>
              task.id === uploadId ? { ...task, progress: percent, status: "uploading" } : task
            )
          );
        };

        xhr.onload = () => {
          if (xhr.status >= 200 && xhr.status < 300) {
            try {
              const body = JSON.parse(xhr.responseText);
              resolve(body.data);
            } catch (e) {
              reject(new Error("解析响应失败"));
            }
          } else {
            try {
              const body = JSON.parse(xhr.responseText);
              reject(new Error(body.error ?? "上传失败"));
            } catch (e) {
              reject(new Error("上传失败"));
            }
          }
        };

        xhr.onerror = () => reject(new Error("网络错误"));

        const formData = new FormData();
        formData.append("file", file, file.name);
        xhr.send(formData);
      });

      setUploads((current) =>
        current.map((task) =>
          task.id === uploadId ? { ...task, progress: 100, status: "success" } : task
        )
      );
      await queryClient.invalidateQueries({ queryKey: ["files"] });
    } catch (err) {
      const message = err instanceof Error ? err.message : "上传失败";
      setUploads((current) =>
        current.map((task) =>
          task.id === uploadId ? { ...task, status: "error", error: message } : task
        )
      );
    }
  };

  async function handleSelected(event: ChangeEvent<HTMLInputElement>) {
    const selection = event.target.files;
    if (!selection) return;
    Array.from(selection).forEach(file => void startUpload(file));
    event.target.value = "";
  }

  function handleDownload(file: FileRecord) {
    // 使用 fetch 获取下载链接或 blob，这里简化为直接打开链接
    // 注意：如果是私有文件，通常需要预签名 URL 或通过 API 代理
    // 对于 DropLite，下载 API 同样需要 Bearer Token。
    // 由于浏览器直接打开 URL 不支持自定义 Header，这里可以使用带 Token 的 URL 参数（如果后端支持）
    // 或者使用 fetch 下载 Blob。
    // 鉴于 MVP 简单性，这里暂时假设后端仅验证 Header，对于 window.open 暂时无法传递 Header。
    // 这是一个已知限制。为了解决这个问题，我们可以改为 fetch 下载 blob。

    fetch(`${FILES_ENDPOINT}/${file.id}/download`, {
      headers: getAuthHeaders()
    })
      .then(res => {
        if (res.ok) return res.blob();
        throw new Error("下载失败");
      })
      .then(blob => {
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = file.original_name;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
      })
      .catch(err => alert(err.message));
  }

  async function handleDelete(file: FileRecord) {
    if (!confirm(`确定删除 ${file.original_name}?`)) return;
    setDeletingIds(prev => new Set(prev).add(file.id));
    try {
      const res = await fetch(`${FILES_ENDPOINT}/${file.id}`, {
        method: "DELETE",
        headers: getAuthHeaders()
      });
      if (!res.ok) throw new Error("删除失败");
      await queryClient.invalidateQueries({ queryKey: ["files"] });
    } catch (err) {
      alert(err instanceof Error ? err.message : "未知错误");
    } finally {
      setDeletingIds(prev => {
        const next = new Set(prev);
        next.delete(file.id);
        return next;
      });
    }
  }

  return (
    <main className="app-shell" style={{ maxWidth: 800, margin: "0 auto", padding: "2rem" }}>
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h1>DropLite</h1>
          <p>轻量文件上传服务</p>
        </div>
        <div style={{ textAlign: 'right' }}>
          <div style={{ marginBottom: 8 }}>{user?.email}</div>
          <button onClick={() => supabase.auth.signOut()}>退出登录</button>
        </div>
      </header>

      <section style={{ marginTop: "2rem" }}>
        <input type="file" multiple onChange={handleSelected} />
      </section>

      {/* Upload List */}
      {uploads.length > 0 && (
        <ul style={{ marginTop: '1rem', listStyle: 'none', padding: 0 }}>
          {uploads.map(task => (
            <li key={task.id} style={{ marginBottom: '0.5rem', border: '1px solid #eee', padding: '0.5rem' }}>
              <div>{task.filename} - {task.status} {task.progress}%</div>
              {task.error && <div style={{ color: 'red' }}>{task.error}</div>}
            </li>
          ))}
        </ul>
      )}

      {/* File List */}
      <section style={{ marginTop: "2rem" }}>
        <h2>文件列表</h2>
        {isLoading && <p>加载中...</p>}
        <ul style={{ listStyle: 'none', padding: 0 }}>
          {sortedFiles.map(file => (
            <li key={file.id} style={{ padding: '1rem', borderBottom: '1px solid #eee', display: 'flex', justifyContent: 'space-between' }}>
              <div>
                <strong>{file.original_name}</strong>
                <div>{formatBytes(file.size_bytes)} - {new Date(file.created_at).toLocaleString()}</div>
              </div>
              <div style={{ gap: '0.5rem', display: 'flex' }}>
                <button onClick={() => handleDownload(file)}>下载</button>
                <button onClick={() => handleDelete(file)} disabled={deletingIds.has(file.id)}>删除</button>
              </div>
            </li>
          ))}
        </ul>
      </section>
    </main>
  );
}

// 鉴权包装器：处理 Loading 和 Login/Main 切换
function AuthWrapper() {
  const { session, loading } = useAuth();

  if (loading) {
    return <div style={{ display: 'flex', justifyContent: 'center', marginTop: '50px' }}>加载中...</div>;
  }

  if (!session) {
    return <LoginPage />;
  }

  return <MainApp />;
}

export default function App() {
  return (
    <AuthProvider>
      <AuthWrapper />
    </AuthProvider>
  );
}
