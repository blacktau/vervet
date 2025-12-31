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
	export interface Result__vervet_internal_settings_Settings_ {
	    isSuccess: boolean;
	    data?: settings.Settings;
	    error: string;
	}
	export interface Result___string_ {
	    isSuccess: boolean;
	    data: string[];
	    error: string;
	}
	export interface Result___vervet_internal_servers_RegisteredServer_ {
	    isSuccess: boolean;
	    data: servers.RegisteredServer[];
	    error: string;
	}
	export interface Result___vervet_internal_settings_Font_ {
	    isSuccess: boolean;
	    data: settings.Font[];
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
	export interface Result_vervet_internal_connections_Connection_ {
	    isSuccess: boolean;
	    // Go type: connections
	    data: any;
	    error: string;
	}
	export interface Result_vervet_internal_servers_RegisteredServer_ {
	    isSuccess: boolean;
	    data: servers.RegisteredServer;
	    error: string;
	}
	export interface Result_vervet_internal_settings_Settings_ {
	    isSuccess: boolean;
	    data: settings.Settings;
	    error: string;
	}
	export interface Result_vervet_internal_settings_WindowState_ {
	    isSuccess: boolean;
	    data: settings.WindowState;
	    error: string;
	}

}

export namespace servers {

	export interface RegisteredServer {
	    id: string;
	    name: string;
	    isGroup: boolean;
	    parentID?: string;
	    color: string;
	    isCluster: boolean;
	    isSrv: boolean;
	}

}

export namespace settings {

	export interface FontSettings {
	    family: string;
	    size: number;
	    name: string;
	}
	export interface EditorSettings {
	    lineNumbers: boolean;
      showFolding: boolean;
      dropText: boolean;
      links: boolean;
	    font: FontSettings;
	}
	export interface Font {
	    name: string;
	    path: string;
	}

	export interface GeneralSettings {
	    theme: string;
	    language: string;
	    font: FontSettings;
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

