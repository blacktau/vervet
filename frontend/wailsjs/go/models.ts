export namespace common {
	
	export class EmptyResult {
	    isSuccess: boolean;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new EmptyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isSuccess = source["isSuccess"];
	        this.error = source["error"];
	    }
	}
	export class Result___vervet_internal_configuration_RegisteredServer_ {
	    isSuccess: boolean;
	    data: configuration.RegisteredServer[];
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new Result___vervet_internal_configuration_RegisteredServer_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isSuccess = source["isSuccess"];
	        this.data = this.convertValues(source["data"], configuration.RegisteredServer);
	        this.error = source["error"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace configuration {
	
	export class RegisteredServer {
	    id: number;
	    name: string;
	    parentId: number;
	    isFolder: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RegisteredServer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.parentId = source["parentId"];
	        this.isFolder = source["isFolder"];
	    }
	}

}

export namespace connections {
	
	export class ActiveConnection {
	
	
	    static createFrom(source: any = {}) {
	        return new ActiveConnection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

