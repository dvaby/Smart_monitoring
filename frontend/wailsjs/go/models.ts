export namespace wailsbind {
	
	export class State {
	    adminConfigured: boolean;
	    authenticated: boolean;
	    telegramConfigured: boolean;
	    proxyConfigured: boolean;
	    chatId: string;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.adminConfigured = source["adminConfigured"];
	        this.authenticated = source["authenticated"];
	        this.telegramConfigured = source["telegramConfigured"];
	        this.proxyConfigured = source["proxyConfigured"];
	        this.chatId = source["chatId"];
	    }
	}

}

