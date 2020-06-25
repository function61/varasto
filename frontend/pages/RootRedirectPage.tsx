import { navigateTo } from 'f61ui/browserutils';
import * as React from 'react';
import * as r from 'generated/frontend_uiroutes';
import { Result } from 'f61ui/component/result';
import { RootFolderId } from 'generated/stoserver/stoservertypes_types';
import { getKeyEncryptionKeys } from 'generated/stoserver/stoservertypes_endpoints';

interface RootRedirectPageState {
	setupCheck: Result<void>;
}

// redirects user to either the app or to the setup wizard (if things have not been set up)
export default class RootRedirectPage extends React.Component<{}, RootRedirectPageState> {
	state: RootRedirectPageState = {
		setupCheck: new Result<void>((_) => {
			this.setState({ setupCheck: _ });
		}),
	};

	componentDidMount() {
		this.fetchData();
	}

	render() {
		// user should not be able to see this, or at least it should flash really fast
		return this.state.setupCheck.draw(() => <h1>Redirecting</h1>);
	}

	private fetchData() {
		// as a setup check, see if user has KEKs set up
		this.state.setupCheck.load(async () => {
			if ((await getKeyEncryptionKeys()).length > 0) {
				// Varasto has been set up
				navigateTo(r.browseUrl({ dir: RootFolderId }));
			} else {
				// Varasto not set up
				navigateTo(r.gettingStartedUrl({ section: 'welcome' }));
			}
		});
	}
}
