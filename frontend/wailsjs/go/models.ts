export namespace cmd {
	
	export class Explanation {
	    flag: string;
	    value: string;
	    desc: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new Explanation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.flag = source["flag"];
	        this.value = source["value"];
	        this.desc = source["desc"];
	        this.note = source["note"];
	    }
	}
	export class ParsedCommand {
	    executable: string;
	    modelPath: string;
	    params: Record<string, string>;
	    extraArgs: string[];
	
	    static createFrom(source: any = {}) {
	        return new ParsedCommand(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.executable = source["executable"];
	        this.modelPath = source["modelPath"];
	        this.params = source["params"];
	        this.extraArgs = source["extraArgs"];
	    }
	}

}

export namespace instance {
	
	export class InstanceState {
	    instanceId: string;
	    status: string;
	    pid: number;
	    // Go type: time
	    startedAt: any;
	    lastError: string;
	
	    static createFrom(source: any = {}) {
	        return new InstanceState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.instanceId = source["instanceId"];
	        this.status = source["status"];
	        this.pid = source["pid"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.lastError = source["lastError"];
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
	export class LlamaInstance {
	    id: string;
	    name: string;
	    runtimeId: string;
	    modelPath: string;
	    params: Record<string, string>;
	    extraArgs: string[];
	    workDir: string;
	    autoStart: boolean;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new LlamaInstance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.runtimeId = source["runtimeId"];
	        this.modelPath = source["modelPath"];
	        this.params = source["params"];
	        this.extraArgs = source["extraArgs"];
	        this.workDir = source["workDir"];
	        this.autoStart = source["autoStart"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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
	export class LogLine {
	    // Go type: time
	    time: any;
	    line: string;
	    stream: string;
	
	    static createFrom(source: any = {}) {
	        return new LogLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = this.convertValues(source["time"], null);
	        this.line = source["line"];
	        this.stream = source["stream"];
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

export namespace runtime {
	
	export class Asset {
	    name: string;
	    size: number;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new Asset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.size = source["size"];
	        this.url = source["url"];
	    }
	}
	export class LlamaRuntime {
	    id: string;
	    buildTag: string;
	    backend: string;
	    variant: string;
	    arch: string;
	    executable: string;
	    workDir: string;
	    source: string;
	    // Go type: time
	    installedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new LlamaRuntime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.buildTag = source["buildTag"];
	        this.backend = source["backend"];
	        this.variant = source["variant"];
	        this.arch = source["arch"];
	        this.executable = source["executable"];
	        this.workDir = source["workDir"];
	        this.source = source["source"];
	        this.installedAt = this.convertValues(source["installedAt"], null);
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
	export class OrphanDir {
	    name: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new OrphanDir(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	    }
	}
	export class ParamDef {
	    flag: string;
	    alias: string;
	    desc: string;
	    hasValue: boolean;
	    group: string;
	
	    static createFrom(source: any = {}) {
	        return new ParamDef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.flag = source["flag"];
	        this.alias = source["alias"];
	        this.desc = source["desc"];
	        this.hasValue = source["hasValue"];
	        this.group = source["group"];
	    }
	}
	export class Release {
	    tagName: string;
	    name: string;
	    // Go type: time
	    publishedAt: any;
	    prerelease: boolean;
	    assets: Asset[];
	
	    static createFrom(source: any = {}) {
	        return new Release(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tagName = source["tagName"];
	        this.name = source["name"];
	        this.publishedAt = this.convertValues(source["publishedAt"], null);
	        this.prerelease = source["prerelease"];
	        this.assets = this.convertValues(source["assets"], Asset);
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

