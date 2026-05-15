export namespace app {
	
	export class UIState {
	    gameExePath: string;
	    modsRoot: string;
	    configOK: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UIState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gameExePath = source["gameExePath"];
	        this.modsRoot = source["modsRoot"];
	        this.configOK = source["configOK"];
	    }
	}

}

export namespace mods {
	
	export class ModVersionRef {
	    folderName: string;
	    manifestFile: string;
	    disabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ModVersionRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folderName = source["folderName"];
	        this.manifestFile = source["manifestFile"];
	        this.disabled = source["disabled"];
	    }
	}
	export class ModManifest {
	    id: string;
	    name: string;
	    author: string;
	    description: string;
	    version: string;
	    has_pck: boolean;
	    has_dll: boolean;
	    dependencies: string[];
	    affects_gameplay: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ModManifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.author = source["author"];
	        this.description = source["description"];
	        this.version = source["version"];
	        this.has_pck = source["has_pck"];
	        this.has_dll = source["has_dll"];
	        this.dependencies = source["dependencies"];
	        this.affects_gameplay = source["affects_gameplay"];
	    }
	}
	export class InstalledMod {
	    folderName: string;
	    manifestFile: string;
	    disabled: boolean;
	    manifest: ModManifest;
	    idUnique: boolean;
	    conflictWith: string[];
	    missingDependencies: string[];
	    available: boolean;
	    layoutNormalized: boolean;
	    alternateVersions: ModVersionRef[];
	
	    static createFrom(source: any = {}) {
	        return new InstalledMod(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folderName = source["folderName"];
	        this.manifestFile = source["manifestFile"];
	        this.disabled = source["disabled"];
	        this.manifest = this.convertValues(source["manifest"], ModManifest);
	        this.idUnique = source["idUnique"];
	        this.conflictWith = source["conflictWith"];
	        this.missingDependencies = source["missingDependencies"];
	        this.available = source["available"];
	        this.layoutNormalized = source["layoutNormalized"];
	        this.alternateVersions = this.convertValues(source["alternateVersions"], ModVersionRef);
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
	export class ModEditPayload {
	    folderName: string;
	    newFolderName: string;
	    layoutNormalized: boolean;
	    manifestFile: string;
	    id: string;
	    name: string;
	    description: string;
	    affects_gameplay: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ModEditPayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.folderName = source["folderName"];
	        this.newFolderName = source["newFolderName"];
	        this.layoutNormalized = source["layoutNormalized"];
	        this.manifestFile = source["manifestFile"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.affects_gameplay = source["affects_gameplay"];
	    }
	}
	
	
	export class ModsOverview {
	    modsDir: string;
	    mods: InstalledMod[];
	    duplicateIDs: string[];
	
	    static createFrom(source: any = {}) {
	        return new ModsOverview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modsDir = source["modsDir"];
	        this.mods = this.convertValues(source["mods"], InstalledMod);
	        this.duplicateIDs = source["duplicateIDs"];
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
	export class NormalizeReport {
	    migrated: string[];
	    skipped: string[];
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new NormalizeReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.migrated = source["migrated"];
	        this.skipped = source["skipped"];
	        this.errors = source["errors"];
	    }
	}

}

