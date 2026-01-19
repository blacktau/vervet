export namespace api {
	
	export enum OperatingSystem {
	    WINDOWS = "windows",
	    LINUX = "linux",
	    OSX = "darwin",
	}
	export interface EmptyResult {
	    isSuccess: boolean;
	    error: string;
	}
	export interface Result__vervet_internal_models_Settings_ {
	    isSuccess: boolean;
	    data?: models.Settings;
	    error: string;
	}
	export interface Result___string_ {
	    isSuccess: boolean;
	    data: string[];
	    error: string;
	}
	export interface Result___vervet_internal_models_Connection_ {
	    isSuccess: boolean;
	    data: models.Connection[];
	    error: string;
	}
	export interface Result___vervet_internal_models_Font_ {
	    isSuccess: boolean;
	    data: models.Font[];
	    error: string;
	}
	export interface Result___vervet_internal_models_RegisteredServer_ {
	    isSuccess: boolean;
	    data: models.RegisteredServer[];
	    error: string;
	}
	export interface Result_string_ {
	    isSuccess: boolean;
	    data: string;
	    error: string;
	}
	export interface Result_vervet_internal_api_OperatingSystem_ {
	    isSuccess: boolean;
	    data: OperatingSystem;
	    error: string;
	}
	export interface Result_vervet_internal_models_Connection_ {
	    isSuccess: boolean;
	    data: models.Connection;
	    error: string;
	}
	export interface Result_vervet_internal_models_RegisteredServer_ {
	    isSuccess: boolean;
	    data: models.RegisteredServer;
	    error: string;
	}
	export interface Result_vervet_internal_models_Settings_ {
	    isSuccess: boolean;
	    data: models.Settings;
	    error: string;
	}
	export interface Result_vervet_internal_models_WindowState_ {
	    isSuccess: boolean;
	    data: models.WindowState;
	    error: string;
	}

}

export namespace models {
	
	export interface Connection {
	    serverID?: string;
	    name?: string;
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
	export interface RegisteredServer {
	    id: string;
	    name: string;
	    parentID?: string;
	    colour: string;
	    isGroup: boolean;
	    isCluster: boolean;
	    isSrv: boolean;
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
	}
	
	
	export interface WindowState {
	    width: number;
	    height: number;
	    x: number;
	    y: number;
	}

}

