import { Panel, Well } from 'f61ui/component/bootstrap';
import { CommandIcon } from 'f61ui/component/CommandButton';
import { Info } from 'f61ui/component/info';
import { Loading } from 'f61ui/component/loading';
import { SecretReveal } from 'f61ui/component/secretreveal';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import { ConfigSetTheMovieDbApikey } from 'generated/stoserver/stoservertypes_commands';
import { getConfig } from 'generated/stoserver/stoservertypes_endpoints';
import { CfgTheMovieDbApikey, ConfigValue } from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface ContentMetadataPageState {
	apikey?: ConfigValue;
}

export default class ContentMetadataPage extends React.Component<{}, ContentMetadataPageState> {
	state: ContentMetadataPageState = {};

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		return (
			<SettingsLayout title="Content metadata" breadcrumbs={[]}>
				<h2>Content metadata providers</h2>

				<Panel
					heading={
						<div>
							TMDb (
							<a href="https://www.themoviedb.org/" target="_blank">
								themoviedb.org
							</a>
							) &nbsp;
							<Info text="For fetching metadata (plot descriptions, poster images etc.) for movies and TV series. This is not required, but if given you can get richer metadata." />
						</div>
					}>
					{this.renderApikeyForm()}
					<Well>
						More info about getting an API key{' '}
						<a href="https://www.themoviedb.org/faq/api" target="_blank">
							here
						</a>
						. It's free and easy.
					</Well>
				</Panel>
			</SettingsLayout>
		);
	}

	private renderApikeyForm() {
		const apikey = this.state.apikey;

		if (!apikey) {
			return <Loading />;
		}

		return (
			<div className="form-horizontal">
				<div className="form-group">
					<label className="col-sm-2 control-label">
						API key
						<CommandIcon command={ConfigSetTheMovieDbApikey()} />
					</label>
					<div className="col-sm-10">
						{apikey.Value !== '' ? (
							<SecretReveal secret={apikey.Value} />
						) : (
							'Not set - unable to fetch metadata'
						)}
					</div>
				</div>
			</div>
		);
	}

	private async fetchData() {
		const apikey = await getConfig(CfgTheMovieDbApikey);

		this.setState({ apikey });
	}
}
