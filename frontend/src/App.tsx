import { useState, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { AuthProvider, useAuth } from "./components/AuthProvider";
import LoginPage from "./pages/LoginPage";
import { LogOut, Upload, FileText, Trash2, Download, Package, Loader2, AlertCircle } from "lucide-react";

// Types
interface FileRecord {
  id: string;
  original_name: string;
  size_bytes: number;
  mime_type: string;
  status: string;
  created_at: string;
}

const API_BASE = (import.meta.env.VITE_API_BASE ?? "").replace(/\/$/, "") || "http://localhost:8080";

function formatBytes(bytes: number, decimals = 2) {
  if (!+bytes) return '0 Bytes';
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`;
}

export default function App() {
  return (
    <AuthProvider>
      <AuthWrapper />
    </AuthProvider>
  );
}

function AuthWrapper() {
  const { session, loading } = useAuth();

  if (loading) {
    return (
      <div className="auth-container">
        <Loader2 className="animate-spin" size={48} color="#6366f1" />
      </div>
    );
  }

  if (!session) {
    return <LoginPage />;
  }

  return <MainApp />;
}

function MainApp() {
  const { session, signOut } = useAuth();
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const queryClient = useQueryClient();

  // Fetch Files
  const { data: files, isLoading, isError } = useQuery<FileRecord[]>({
    queryKey: ["files"],
    queryFn: async () => {
      const res = await fetch(`${API_BASE}/files`, {
        headers: {
          Authorization: `Bearer ${session?.access_token}`,
        },
      });
      if (!res.ok) throw new Error("Failed to fetch files");
      const json = await res.json();
      return json.data || [];
    },
  });

  // Upload Mutation
  const uploadMutation = useMutation({
    mutationFn: async (file: File) => {
      return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open("POST", `${API_BASE}/files`, true);
        xhr.setRequestHeader("Authorization", `Bearer ${session?.access_token}`);

        xhr.upload.onprogress = (e) => {
          if (e.lengthComputable) {
            setProgress(Math.round((e.loaded / e.total) * 100));
          }
        };

        xhr.onload = () => {
          if (xhr.status >= 200 && xhr.status < 300) {
            resolve(JSON.parse(xhr.response));
          } else {
            reject(new Error(xhr.statusText || "Upload failed"));
          }
        };

        xhr.onerror = () => reject(new Error("Network error"));

        const formData = new FormData();
        formData.append("file", file);
        xhr.send(formData);
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["files"] });
      setUploading(false);
      setProgress(0);
      if (fileInputRef.current) fileInputRef.current.value = "";
    },
    onError: (err) => {
      console.error(err);
      setUploading(false);
      alert("上传失败: " + err);
    },
  });

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setUploading(true);
      uploadMutation.mutate(e.target.files[0]);
    }
  };

  const handleDownload = async (id: string, name: string) => {
    try {
      const res = await fetch(`${API_BASE}/files/${id}/download`, {
        headers: {
          Authorization: `Bearer ${session?.access_token}`,
        },
      });
      if (!res.ok) throw new Error("Download failed");
      const blob = await res.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = name;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      a.remove();
    } catch (error) {
      console.error(error);
      alert("下载失败");
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("确定要删除这个文件吗？")) return;
    try {
      const res = await fetch(`${API_BASE}/files/${id}`, {
        method: "DELETE",
        headers: {
          Authorization: `Bearer ${session?.access_token}`,
        },
      });
      if (!res.ok) throw new Error("Delete failed");
      queryClient.invalidateQueries({ queryKey: ["files"] });
    } catch (error) {
      console.error(error);
      alert("删除失败");
    }
  };

  return (
    <div>
      <header className="app-header">
        <div className="container" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', height: '100%' }}>
          <div className="logo">
            <Package size={28} color="#818cf8" />
            <span>DropLite</span>
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
            <span style={{ fontSize: '0.9rem', color: 'var(--text-secondary)' }}>
              {session?.user?.email}
            </span>
            <button className="btn btn-secondary" onClick={() => signOut()}>
              <LogOut size={16} /> 退出
            </button>
          </div>
        </div>
      </header>

      <main className="container">
        {/* Upload Section */}
        <div className="card">
          <div
            className={`upload-zone ${uploading ? 'dragging' : ''}`}
            onClick={() => !uploading && fileInputRef.current?.click()}
          >
            <input
              type="file"
              ref={fileInputRef}
              onChange={handleFileChange}
              style={{ display: "none" }}
              disabled={uploading}
            />

            {uploading ? (
              <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '1rem' }}>
                <Loader2 className="animate-spin" size={48} color="#6366f1" />
                <div style={{ width: '100%', maxWidth: '300px', background: 'rgba(255,255,255,0.1)', borderRadius: '10px', overflow: 'hidden', height: '8px' }}>
                  <div style={{ width: `${progress}%`, background: '#6366f1', height: '100%', transition: 'width 0.2s' }} />
                </div>
                <p>正在上传... {progress}%</p>
              </div>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '1rem' }}>
                <div style={{ background: 'rgba(99, 102, 241, 0.1)', padding: '1rem', borderRadius: '50%' }}>
                  <Upload size={32} color="#6366f1" />
                </div>
                <div>
                  <h3 style={{ marginBottom: '0.5rem' }}>点击或拖拽文件到这里上传</h3>
                  <p>支持最大 5GB 文件</p>
                </div>
                <button className="btn btn-primary">选择文件</button>
              </div>
            )}
          </div>
        </div>

        {/* File List */}
        <div className="card">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem' }}>
            <h2>我的文件</h2>
            <div style={{ fontSize: '0.9rem', color: 'var(--text-secondary)' }}>
              {files?.length || 0} 个文件
            </div>
          </div>

          {isLoading ? (
            <div style={{ textAlign: 'center', padding: '2rem' }}>
              <Loader2 className="animate-spin" size={24} color="#6366f1" />
            </div>
          ) : isError ? (
            <div style={{ color: 'var(--danger)', textAlign: 'center', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '0.5rem' }}>
              <AlertCircle size={20} /> 加载失败
            </div>
          ) : files?.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '3rem', color: 'var(--text-secondary)' }}>
              <p>暂无文件，快去上传一个吧！</p>
            </div>
          ) : (
            <div className="file-list">
              {files?.map((file) => (
                <div key={file.id} className="file-item">
                  <div className="file-icon">
                    <FileText size={24} />
                  </div>

                  <div style={{ minWidth: 0 }}>
                    <div className="file-name" title={file.original_name}>
                      {file.original_name}
                    </div>
                    <div className="file-meta">
                      {new Date(file.created_at).toLocaleString()}
                    </div>
                  </div>

                  <div style={{ color: 'var(--text-secondary)', fontSize: '0.9rem', whiteSpace: 'nowrap' }}>
                    {formatBytes(file.size_bytes)}
                  </div>

                  <div style={{ display: 'flex', gap: '0.5rem' }}>
                    <button
                      className="btn btn-icon"
                      onClick={() => handleDownload(file.id, file.original_name)}
                      title="下载"
                    >
                      <Download size={18} />
                    </button>
                    <button
                      className="btn btn-icon"
                      onClick={() => handleDelete(file.id)}
                      title="删除"
                      style={{ color: 'var(--danger)' }}
                    >
                      <Trash2 size={18} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
