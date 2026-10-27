export namespace main {
	
	export class ApacheSettings {
	    publicationName: string;
	    address: string;
	    port: string;
	    confPath: string;
	    warningDuration: string;
	
	    static createFrom(source: any = {}) {
	        return new ApacheSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.publicationName = source["publicationName"];
	        this.address = source["address"];
	        this.port = source["port"];
	        this.confPath = source["confPath"];
	        this.warningDuration = source["warningDuration"];
	    }
	}
	export class DataMobileSettings {
	    user: string;
	    pass: string;
	
	    static createFrom(source: any = {}) {
	        return new DataMobileSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user = source["user"];
	        this.pass = source["pass"];
	    }
	}
	export class LmczSettings {
	    host: string;
	    subnet: string;
	    port: string;
	    scannerTimeout: string;
	    user: string;
	    pass: string;
	    token: string;
	
	    static createFrom(source: any = {}) {
	        return new LmczSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.subnet = source["subnet"];
	        this.port = source["port"];
	        this.scannerTimeout = source["scannerTimeout"];
	        this.user = source["user"];
	        this.pass = source["pass"];
	        this.token = source["token"];
	    }
	}
	export class LmczStatusResult {
	    status: string;
	    httpCode: number;
	
	    static createFrom(source: any = {}) {
	        return new LmczStatusResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.httpCode = source["httpCode"];
	    }
	}
	export class Settings {
	    configPath: string;
	    configSaved: boolean;
	    clientRequestTimeout: string;
	    apache: ApacheSettings;
	    datamobile: DataMobileSettings;
	    lmcz: LmczSettings;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.configPath = source["configPath"];
	        this.configSaved = source["configSaved"];
	        this.clientRequestTimeout = source["clientRequestTimeout"];
	        this.apache = this.convertValues(source["apache"], ApacheSettings);
	        this.datamobile = this.convertValues(source["datamobile"], DataMobileSettings);
	        this.lmcz = this.convertValues(source["lmcz"], LmczSettings);
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

