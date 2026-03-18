export namespace monitor {
	
	export class Action {
	    type: string;
	    target: string;
	    result?: string;
	    timestamp: number;
	
	    static createFrom(source: any = {}) {
	        return new Action(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.target = source["target"];
	        this.result = source["result"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class Session {
	    pid: number;
	    sessionId: string;
	    cwd: string;
	    startedAt: number;
	    source: string;
	    projectName: string;
	    status: string;
	    duration: string;
	    recentActions: Action[];
	
	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.sessionId = source["sessionId"];
	        this.cwd = source["cwd"];
	        this.startedAt = source["startedAt"];
	        this.source = source["source"];
	        this.projectName = source["projectName"];
	        this.status = source["status"];
	        this.duration = source["duration"];
	        this.recentActions = this.convertValues(source["recentActions"], Action);
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

