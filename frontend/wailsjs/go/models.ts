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
	export interface Result___int_ {
	    isSuccess: boolean;
	    data: number[];
	    error: string;
	}
	export interface Result___vervet_internal_configuration_RegisteredServer_ {
	    isSuccess: boolean;
	    data: configuration.RegisteredServer[];
	    error: string;
	}
	export interface Result_vervet_internal_api_OperatingSystem_ {
	    isSuccess: boolean;
	    data: OperatingSystem;
	    error: string;
	}

}

export namespace configuration {
	
	export interface RegisteredServer {
	    id: number;
	    name: string;
	    parentId: number;
	    isGroup: boolean;
	}

}

