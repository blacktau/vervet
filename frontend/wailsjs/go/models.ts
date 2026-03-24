export namespace api {
	
	export enum OperatingSystem {
	    WINDOWS = "windows",
	    LINUX = "linux",
	    OSX = "darwin",
	}
	export interface EmptyResult {
	    isSuccess: boolean;
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface FileFilter {
	    displayName: string;
	    pattern: string;
	}
	export interface Result__vervet_internal_models_Settings_ {
	    isSuccess: boolean;
	    data?: models.Settings;
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result___string_ {
	    isSuccess: boolean;
	    data: string[];
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result___vervet_internal_models_Connection_ {
	    isSuccess: boolean;
	    data: models.Connection[];
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result___vervet_internal_models_DirectoryEntry_ {
	    isSuccess: boolean;
	    data: models.DirectoryEntry[];
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result___vervet_internal_models_Font_ {
	    isSuccess: boolean;
	    data: models.Font[];
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result___vervet_internal_models_Index_ {
	    isSuccess: boolean;
	    data: models.Index[];
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result___vervet_internal_models_RegisteredServer_ {
	    isSuccess: boolean;
	    data: models.RegisteredServer[];
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result_bool_ {
	    isSuccess: boolean;
	    data: boolean;
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result_map_string_interface____ {
	    isSuccess: boolean;
	    data: Record<string, any>;
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result_string_ {
	    isSuccess: boolean;
	    data: string;
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result_vervet_internal_api_OperatingSystem_ {
	    isSuccess: boolean;
	    data: OperatingSystem;
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result_vervet_internal_models_CollectionSchema_ {
	    isSuccess: boolean;
	    data: models.CollectionSchema;
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result_vervet_internal_models_ConnectionConfig_ {
	    isSuccess: boolean;
	    data: models.ConnectionConfig;
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result_vervet_internal_models_Connection_ {
	    isSuccess: boolean;
	    data: models.Connection;
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result_vervet_internal_models_QueryResult_ {
	    isSuccess: boolean;
	    data: models.QueryResult;
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result_vervet_internal_models_RegisteredServer_ {
	    isSuccess: boolean;
	    data: models.RegisteredServer;
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result_vervet_internal_models_Settings_ {
	    isSuccess: boolean;
	    data: models.Settings;
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result_vervet_internal_models_WindowState_ {
	    isSuccess: boolean;
	    data: models.WindowState;
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result_vervet_internal_models_WorkspaceData_ {
	    isSuccess: boolean;
	    data: models.WorkspaceData;
	    errorCode?: string;
	    errorDetail?: string;
	}
	export interface Result_vervet_internal_models_Workspace_ {
	    isSuccess: boolean;
	    data: models.Workspace;
	    errorCode?: string;
	    errorDetail?: string;
	}

}

export namespace models {
	
	export interface FieldInfo {
	    path: string;
	    types: string[];
	    children: FieldInfo[];
	}
	export interface CollectionSchema {
	    fields: FieldInfo[];
	}
	export interface Connection {
	    serverID?: string;
	    name?: string;
	}
	export interface OIDCConfig {
	    providerUrl: string;
	    clientId: string;
	    scopes?: string[];
	    workloadIdentity: boolean;
	}
	export interface ConnectionConfig {
	    uri: string;
	    authMethod: string;
	    oidcConfig?: OIDCConfig;
	    refreshToken?: string;
	}
	export interface IndexKeyField {
	    field: string;
	    direction: any;
	}
	export interface CreateIndexRequest {
	    keys: IndexKeyField[];
	    name?: string;
	    unique: boolean;
	    sparse: boolean;
	    ttl?: number;
	}
	export interface DirectoryEntry {
	    name: string;
	    path: string;
	    isDirectory: boolean;
	    children?: DirectoryEntry[];
	}
	export interface EditIndexRequest {
	    oldName: string;
	    keys: IndexKeyField[];
	    name?: string;
	    unique: boolean;
	    sparse: boolean;
	    ttl?: number;
	}
	export interface FontSettings {
	    family: string;
	    size: number;
	    name: string;
	}
	export interface EditorSettings {
	    lineNumbers: boolean;
	    font: FontSettings;
	    showFolding: boolean;
	    dropText: boolean;
	    links: boolean;
	    queryEngine: string;
	}
	
	export interface Font {
	    family?: string;
	    isFixedWidth?: boolean;
	}
	
	export interface GeneralSettings {
	    theme: string;
	    language: string;
	    font: FontSettings;
	}
	export interface Index {
	    name: string;
	    keys: IndexKeyField[];
	    unique: boolean;
	    sparse: boolean;
	    ttl?: number;
	    size: number;
	    usage: number;
	}
	
	
	export interface QueryResult {
	    documents: any[];
	    rawOutput: string;
	    operationType?: string;
	    affectedCount?: number;
	}
	export interface RegisteredServer {
	    id: string;
	    name: string;
	    parentID?: string;
	    colour: string;
	    isGroup: boolean;
	    isCluster: boolean;
	    isSrv: boolean;
	}
	export interface WorkspacesSettings {
	    fileExtensions: string[];
	}
	export interface TerminalSettings {
	    font: FontSettings;
	    cursorStyle: string;
	}
	export interface WindowSettings {
	    width: number;
	    height: number;
	    asideWidth: number;
	    maximized: boolean;
	    positionX: number;
	    positionY: number;
	}
	export interface Settings {
	    window: WindowSettings;
	    general: GeneralSettings;
	    editor: EditorSettings;
	    terminal: TerminalSettings;
	    workspaces: WorkspacesSettings;
	}
	
	
	export interface WindowState {
	    width: number;
	    height: number;
	    x: number;
	    y: number;
	}
	export interface Workspace {
	    id: string;
	    name: string;
	    folders: string[];
	}
	export interface WorkspaceData {
	    activeWorkspaceId: string;
	    workspaces: Workspace[];
	}

}

