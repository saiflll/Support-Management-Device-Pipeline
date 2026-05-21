import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import { writable } from 'svelte/store';

export type Agent = {
  id: string;
  name: string;
  provider: string;
  model: string;
  base_url: string | null;
  api_key: string | null;
  capabilities: string[];
  temperature: number | null;
  system_prompt: string | null;
};

export type ChatMessage = {
  role: string;
  content: string;
};

export type FileEntry = {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  modified: string;
};

export type TabType = "terminal" | "ai" | "notepad" | "editor" | "files";

export type Tab = {
  id: string;
  type: TabType;
  title: string;
};

export type PtyOutputEvent = {
  session_id: string;
  data: string;
};

// Workspace layout state
export const activePanel = writable<string>("terminal");
export const activeFile = writable<string | null>(null);
export const fileContent = writable<string>("");
export const fileEntries = writable<FileEntry[]>([]);
export const currentDir = writable<string>("");

// Terminal state
export const terminalSessions = writable<Map<string, any>>(new Map());
export const activeTerminalSessionId = writable<string | null>(null);

// AI state
export const agents = writable<Agent[]>([]);
export const chatMessages = writable<ChatMessage[]>([]);
export const selectedAgent = writable<string>("coder");

// Notepad state
export const notepadContent = writable<string>("");
export const notepadLanguage = writable<string>("typescript");

// Split terminal state
export const splitLayout = writable<string>("horizontal"); // "horizontal" | "vertical" | "quad"

// File explorer state
export function loadDir(dir: string) {
  currentDir.set(dir);
  invoke<FileEntry[]>("fs_list_dir", { path: dir }).then((entries) => {
    fileEntries.set(entries);
  });
}

export function openFile(path: string) {
  activeFile.set(path);
  invoke<string>("fs_read_file", { path }).then((content) => {
    fileContent.set(content);
  });
}

export function saveFile(path: string, content: string) {
  invoke("fs_write_file", { path, content }).then(() => {
    if (activeFile) {
      // refresh
    }
  });
}

// Agent management
export async function loadAgents() {
  const list = await invoke<Agent[]>("ai_list_agents");
  agents.set(list);
}

export function sendChat(agentId: string, messages: ChatMessage[]) {
  return invoke<{ agent_id: string; content: string; provider: string; model: string }>("ai_chat", {
    agentId,
    messages,
  });
}

export function autoRoute(taskType: string) {
  return invoke<Agent[]>("ai_auto_route", { taskType });
}

// CodeEditor languages configuration store
export type LangConfig = {
  id: string;
  name: string;
  description: string;
  extensions: string[];
  iconText: string;
  color: string;
};

export const SUPPORTED_LANGS: LangConfig[] = [
  { id: "typescript", name: "TypeScript / JavaScript", description: "Enables code highlighting & autocompletion for .js, .ts, .jsx, .tsx, .svelte, .vue files.", extensions: ["js", "ts", "jsx", "tsx", "svelte", "vue"], iconText: "JS", color: "#f7df1e" },
  { id: "rust", name: "Rust Lang Support", description: "Enables syntax parsing & cargo tooling integrations for .rs source files.", extensions: ["rs"], iconText: "RS", color: "#ff8243" },
  { id: "python", name: "Python 3 Engine", description: "Enables syntax analysis, indentation styling & keywords for .py scripts.", extensions: ["py"], iconText: "PY", color: "#3572A5" },
  { id: "html", name: "HTML / Svelte / Markup", description: "Enables tag coloring, structure layout parsing for markup & components.", extensions: ["html", "htm", "svelte", "vue"], iconText: "HTML", color: "#ff3e00" },
  { id: "css", name: "CSS Style Sheets", description: "Enables rule coloring, variables parsing & autocompletes for .css.", extensions: ["css"], iconText: "CSS", color: "#38bdf8" },
  { id: "json", name: "JSON & Locks", description: "Enables object brace matching, keys highlighting & maps parsing for configs.", extensions: ["json", "lock"], iconText: "JSON", color: "#10b981" },
  { id: "markdown", name: "Markdown Docs", description: "Enables bold, header styling, list parsing and guides for documentation.", extensions: ["md"], iconText: "MD", color: "#a855f7" },
  { id: "go", name: "Go Lang", description: "Enables keywords highlighting, structure parsing for Go (.go) backend files.", extensions: ["go"], iconText: "GO", color: "#00ADD8" },
  { id: "cpp", name: "C / C++ / C#", description: "Enables high-performance systems highlighting for .c, .cpp, .h, .hpp, .cs files.", extensions: ["c", "cpp", "h", "hpp", "cs"], iconText: "C++", color: "#3b82f6" },
  { id: "java", name: "Java Language", description: "Enables strong type checking highlighting, class structures parsing for .java files.", extensions: ["java"], iconText: "JAVA", color: "#ea2d2e" },
  { id: "php", name: "PHP Server Side", description: "Enables syntax keywords highlighting, tag scripting support for .php pages.", extensions: ["php"], iconText: "PHP", color: "#777bb4" },
  { id: "sql", name: "SQL Databases", description: "Enables database query commands, schema syntax highlighting for .sql scripts.", extensions: ["sql"], iconText: "SQL", color: "#e38c00" },
  { id: "yaml", name: "YAML / TOML", description: "Enables indentation validation, key-value maps parsing for .yaml, .toml configs.", extensions: ["yaml", "yml", "toml"], iconText: "YAML", color: "#fb7185" },
  { id: "xml", name: "XML Docs", description: "Enables structured nodes, attributes & nested tag coloring for .xml files.", extensions: ["xml"], iconText: "XML", color: "#38bdf8" }
];

function getInitialLangs(): Record<string, boolean> {
  const DEFAULT_LANGS = {
    typescript: true,
    rust: true,
    python: true,
    html: true,
    css: true,
    json: true,
    markdown: true,
    go: false,
    cpp: false,
    java: false,
    php: false,
    sql: false,
    yaml: false,
    xml: false
  };
  try {
    const stored = localStorage.getItem("contlib-editor-langs");
    if (stored) {
      return { ...DEFAULT_LANGS, ...JSON.parse(stored) };
    }
  } catch {}
  return DEFAULT_LANGS;
}

export const editorLanguages = writable<Record<string, boolean>>(getInitialLangs());

export function saveEditorLangs(langs: Record<string, boolean>) {
  editorLanguages.set(langs);
  try {
    localStorage.setItem("contlib-editor-langs", JSON.stringify(langs));
  } catch (e) {
    console.error("Failed to save editor languages to localStorage:", e);
  }
}
