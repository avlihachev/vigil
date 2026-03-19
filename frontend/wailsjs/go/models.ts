export namespace main {
	
	export class Settings {
	    notifyConfirm: boolean;
	    notifyWaiting: boolean;
	    badgeConfirm: boolean;
	    badgeWaiting: boolean;
	    badgeActive: boolean;
	    lastUpdateCheck?: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.notifyConfirm = source["notifyConfirm"];
	        this.notifyWaiting = source["notifyWaiting"];
	        this.badgeConfirm = source["badgeConfirm"];
	        this.badgeWaiting = source["badgeWaiting"];
	        this.badgeActive = source["badgeActive"];
	        this.lastUpdateCheck = source["lastUpdateCheck"];
	    }
	}

}

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
	export class HistoricalSession {
	    sessionId: string;
	    name: string;
	    lastActiveAt: number;
	    tokensIn: string;
	    tokensOut: string;
	
	    static createFrom(source: any = {}) {
	        return new HistoricalSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.name = source["name"];
	        this.lastActiveAt = source["lastActiveAt"];
	        this.tokensIn = source["tokensIn"];
	        this.tokensOut = source["tokensOut"];
	    }
	}
	export class ProjectHistory {
	    projectName: string;
	    cwd: string;
	    sessions: HistoricalSession[];
	
	    static createFrom(source: any = {}) {
	        return new ProjectHistory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectName = source["projectName"];
	        this.cwd = source["cwd"];
	        this.sessions = this.convertValues(source["sessions"], HistoricalSession);
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
	export class Session {
	    pid: number;
	    sessionId: string;
	    cwd: string;
	    startedAt: number;
	    source: string;
	    projectName: string;
	    name: string;
	    status: string;
	    duration: string;
	    tokensIn: string;
	    tokensOut: string;
	    recentActions: Action[];
	    sibling?: string;
	
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
	        this.name = source["name"];
	        this.status = source["status"];
	        this.duration = source["duration"];
	        this.tokensIn = source["tokensIn"];
	        this.tokensOut = source["tokensOut"];
	        this.recentActions = this.convertValues(source["recentActions"], Action);
	        this.sibling = source["sibling"];
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

