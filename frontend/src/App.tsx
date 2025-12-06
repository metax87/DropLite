import { useMemo, useState, type ChangeEvent } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

const API_BASE = (import.meta.env.VITE_API_BASE ?? "").replace(/\/$/, "");
const API_KEY = import.meta.env.VITE_API_KEY ?? "dev-api-key-123456";
const FILES_ENDPOINT = `${API_BASE}/files`;

// 通用请求头
function getAuthHeaders(): Record<string, string> {
  return {
    Accept: "application/json",
    Authorization: `ApiKey ${API_KEY}`
  };
}

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

async function fetchFiles(): Promise<FileRecord[]> {
  const response = await fetch(FILES_ENDPOINT, {
    headers: getAuthHeaders()
  });
  if (!response.ok) {
    throw new Error("无法获取文件列表");
  }
  const payload = await response.json();
  return payload?.data ?? [];
}

function uploadFile(file: File, onProgress: (value: number) => void) {
  return new Promise<FileRecord>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", FILES_ENDPOINT);
    xhr.setRequestHeader("Authorization", `ApiKey ${API_KEY}`);

    xhr.upload.onprogress = (event) => {
      if (!event.lengthComputable) return;
      const percent = Math.round((event.loaded / event.total) * 100);
      onProgress(percent);
    };

    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          const body = JSON.parse(xhr.responseText) as { data?: FileRecord };
          if (!body?.data) {
            reject(new Error("服务端未返回文件记录"));
            return;
          }
          resolve(body.data);
        } catch (error) {
          reject(new Error("解析服务端响应失败"));
        }
        return;
      }

      try {
        const errorBody = JSON.parse(xhr.responseText);
        reject(new Error(errorBody.error ?? "上传失败"));
      } catch (_error) {
        reject(new Error("上传失败"));
      }
    };

    xhr.onerror = () => reject(new Error("网络错误，请稍后重试"));

    const formData = new FormData();
    formData.append("file", file, file.name);
    xhr.send(formData);
  });
}

async function deleteFile(id: string): Promise<void> {
  const response = await fetch(`${FILES_ENDPOINT}/${id}`, {
    method: "DELETE",
    headers: getAuthHeaders()
  });
  if (!response.ok) {
    const errorBody = await response.json().catch(() => ({}));
    throw new Error(errorBody.error ?? "删除失败");
  }
}

function formatBytes(size: number) {
  if (!Number.isFinite(size) || size <= 0) return "-";
  const units = ["B", "KB", "MB", "GB"];
  const exponent = Math.min(Math.floor(Math.log(size) / Math.log(1024)), units.length - 1);
  const value = size / Math.pow(1024, exponent);
  return `${value.toFixed(1)} ${units[exponent]}`;
}

export default function App() {
  const queryClient = useQueryClient();
  const [uploads, setUploads] = useState<UploadTask[]>([]);
  const [deletingIds, setDeletingIds] = useState<Set<string>>(new Set());
  const { data: files = [], isLoading, isFetching, error } = useQuery({
    queryKey: ["files"],
    queryFn: fetchFiles
  });

  const sortedFiles = useMemo(() => {
    return [...files].sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at));
  }, [files]);

  async function handleSelected(event: ChangeEvent<HTMLInputElement>) {
    const selection = event.target.files;
    if (!selection || selection.length === 0) {
      return;
    }

    const filesToUpload = Array.from(selection);
    event.target.value = "";

    filesToUpload.forEach((file) => {
      void startUpload(file);
    });
  }

  async function startUpload(file: File) {
    const uploadId = crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random()}`;
    setUploads((current) => [
      ...current,
      { id: uploadId, filename: file.name, progress: 0, status: "uploading" }
    ]);

    try {
      await uploadFile(file, (value) => {
        setUploads((current) =>
          current.map((task) =>
            task.id === uploadId ? { ...task, progress: value, status: "uploading" } : task
          )
        );
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
          task.id === uploadId
            ? { ...task, status: "error", error: message }
            : task
        )
      );
    }
  }

  function handleDownload(file: FileRecord) {
    if (file.status !== "stored") {
      alert("文件尚未就绪，无法下载");
      return;
    }
    window.open(`${FILES_ENDPOINT}/${file.id}/download`, "_blank");
  }

  async function handleDelete(file: FileRecord) {
    if (!confirm(`确定要删除「${file.original_name}」吗？`)) {
      return;
    }

    setDeletingIds((current) => new Set(current).add(file.id));

    try {
      await deleteFile(file.id);
      await queryClient.invalidateQueries({ queryKey: ["files"] });
    } catch (err) {
      const message = err instanceof Error ? err.message : "删除失败";
      alert(message);
    } finally {
      setDeletingIds((current) => {
        const next = new Set(current);
        next.delete(file.id);
        return next;
      });
    }
  }

  return (
    <main className="app-shell" style={{ maxWidth: 800, margin: "0 auto", padding: "2rem" }}>
      <header>
        <h1>DropLite</h1>
        <p>轻量文件上传服务的 React 客户端，支持上传、下载和删除。</p>
      </header>

      <section style={{ marginTop: "2rem" }}>
        <label htmlFor="uploader">选择文件（支持多选，单个文件 ≤ 100MB）</label>
        <input id="uploader" type="file" multiple onChange={handleSelected} />
        <p style={{ fontSize: "0.9rem", color: "#555" }}>
          选择后会立即上传，并在下方显示进度。上传完成会自动刷新列表。
        </p>
      </section>

      {uploads.length > 0 && (
        <section style={{ marginTop: "2rem" }}>
          <h2>上传进度</h2>
          <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
            {uploads.map((task) => (
              <li
                key={task.id}
                style={{
                  padding: "0.75rem 0",
                  borderBottom: "1px solid #eee"
                }}
              >
                <div style={{ display: "flex", justifyContent: "space-between" }}>
                  <span>{task.filename}</span>
                  <span>
                    {task.status === "uploading" && `${task.progress}%`}
                    {task.status === "success" && "完成"}
                    {task.status === "error" && "失败"}
                  </span>
                </div>
                <div
                  style={{
                    height: 6,
                    background: "#eee",
                    borderRadius: 4,
                    overflow: "hidden",
                    marginTop: 8
                  }}
                >
                  <div
                    style={{
                      width: `${task.progress}%`,
                      height: "100%",
                      transition: "width 0.2s ease",
                      background:
                        task.status === "error"
                          ? "#d9534f"
                          : task.status === "success"
                            ? "#5cb85c"
                            : "#4285f4"
                    }}
                  />
                </div>
                {task.error && (
                  <p style={{ color: "#d9534f", margin: "0.5rem 0 0" }}>{task.error}</p>
                )}
              </li>
            ))}
          </ul>
        </section>
      )}

      <section style={{ marginTop: "2rem" }}>
        <h2>
          已上传文件 {isFetching && <small style={{ fontSize: "0.85rem" }}>刷新中…</small>}
        </h2>
        {isLoading && <p>加载中…</p>}
        {error && !isLoading && <p style={{ color: "#d9534f" }}>加载失败：{String(error)}</p>}
        {!isLoading && !error && sortedFiles.length === 0 && <p>暂无文件。</p>}
        {!isLoading && !error && sortedFiles.length > 0 && (
          <ul style={{ listStyle: "none", padding: 0 }}>
            {sortedFiles.map((file) => (
              <li
                key={file.id}
                style={{
                  padding: "1rem 0",
                  borderBottom: "1px solid #eee",
                  display: "flex",
                  justifyContent: "space-between",
                  gap: "1rem",
                  alignItems: "center",
                  flexWrap: "wrap"
                }}
              >
                <div style={{ flex: 1, minWidth: 200 }}>
                  <strong>{file.original_name}</strong>
                  <div style={{ fontSize: "0.9rem", color: "#555" }}>
                    {file.mime_type} · {formatBytes(file.size_bytes)}
                  </div>
                  <div style={{ fontSize: "0.85rem", color: "#666", marginTop: 4 }}>
                    状态：{file.status} · {new Date(file.created_at).toLocaleString()}
                  </div>
                </div>
                <div style={{ display: "flex", gap: "0.5rem" }}>
                  <button
                    onClick={() => handleDownload(file)}
                    disabled={file.status !== "stored"}
                    style={{
                      padding: "0.4rem 0.8rem",
                      background: file.status === "stored" ? "#4285f4" : "#ccc",
                      color: "#fff",
                      border: "none",
                      borderRadius: 4,
                      cursor: file.status === "stored" ? "pointer" : "not-allowed"
                    }}
                  >
                    下载
                  </button>
                  <button
                    onClick={() => handleDelete(file)}
                    disabled={deletingIds.has(file.id)}
                    style={{
                      padding: "0.4rem 0.8rem",
                      background: deletingIds.has(file.id) ? "#ccc" : "#d9534f",
                      color: "#fff",
                      border: "none",
                      borderRadius: 4,
                      cursor: deletingIds.has(file.id) ? "not-allowed" : "pointer"
                    }}
                  >
                    {deletingIds.has(file.id) ? "删除中…" : "删除"}
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  );
}
