import { CollapsePanel, Panel } from 'f61ui/component/bootstrap';
import { CommandIcon } from 'f61ui/component/CommandButton';
import { Info } from 'f61ui/component/info';
import { Result } from 'f61ui/component/result';
import { SecretReveal } from 'f61ui/component/secretreveal';
import {
	ConfigSetIgdbApikey,
	ConfigSetTheMovieDBApiKey,
} from 'generated/stoserver/stoservertypes_commands';
import { getConfig } from 'generated/stoserver/stoservertypes_endpoints';
import {
	CfgIgdbApikey,
	CfgTheMovieDbApikey,
	ConfigValue,
} from 'generated/stoserver/stoservertypes_types';
import { AdminLayout } from 'layout/AdminLayout';
import * as React from 'react';

interface ContentMetadataPageState {
	apikeyTmdb: Result<ConfigValue>;
	apikeyIgdb: Result<ConfigValue>;
}

export default class ContentMetadataPage extends React.Component<{}, ContentMetadataPageState> {
	state: ContentMetadataPageState = {
		apikeyTmdb: new Result<ConfigValue>((_) => {
			this.setState({ apikeyTmdb: _ });
		}),
		apikeyIgdb: new Result<ConfigValue>((_) => {
			this.setState({ apikeyIgdb: _ });
		}),
	};

	componentDidMount() {
		this.fetchData();
	}

	componentWillReceiveProps() {
		this.fetchData();
	}

	render() {
		return (
			<AdminLayout title="Content metadata" breadcrumbs={[]}>
				<h2>Content metadata providers</h2>

				<Panel
					heading={
						<div>
							Movies &amp; TV shows - TMDb (
							<a href="https://www.themoviedb.org/" target="_blank">
								themoviedb.org
							</a>
							) &nbsp;
							<Info text="You get: plot descriptions, poster images etc." />
						</div>
					}>
					{this.renderTmdb()}
				</Panel>

				<Panel
					heading={
						<div>
							Games - IGDB (
							<a href="https://www.igdb.com/" target="_blank">
								igdb.com
							</a>
							) &nbsp;
							<Info text="You get: game overview, screenshot, links, YouTube videos etc." />
						</div>
					}>
					{this.renderIgdb()}
				</Panel>

				<CollapsePanel heading="Info" visualStyle="info">
					<p>
						These optional providers enable Varasto to display rich metadata for various
						kinds of content.
					</p>
					<p>
						Unfortunately, you have to register &amp; configure API keys for them first
						(so the providers can protect themselves from abuse).
					</p>
					<p>
						The API keys are <b>free for you to use</b>.
					</p>
				</CollapsePanel>
			</AdminLayout>
		);
	}

	private renderTmdb() {
		const [apikey, loadingOrError] = this.state.apikeyTmdb.unwrap();

		if (!apikey) {
			return loadingOrError;
		}

		return (
			<div className="form-horizontal">
				<div className="form-group">
					<label className="col-sm-2 control-label">
						API key &nbsp;
						<CommandIcon command={ConfigSetTheMovieDBApiKey(apikey.Value)} />
					</label>
					<div className="col-sm-10">
						{apikey.Value !== '' ? (
							<SecretReveal secret={apikey.Value} />
						) : (
							<span>
								Not set - unable to fetch metadata.{' '}
								<a href="https://www.themoviedb.org/faq/api" target="_blank">
									Register for an API key
								</a>
								&nbsp; (&amp; get more info).
							</span>
						)}
					</div>
				</div>
			</div>
		);
	}

	private renderIgdb() {
		const [apikey, loadingOrError] = this.state.apikeyIgdb.unwrap();

		if (!apikey) {
			return loadingOrError;
		}

		return (
			<div className="form-horizontal">
				<div className="form-group">
					<label className="col-sm-2 control-label">
						API key &nbsp;
						<CommandIcon command={ConfigSetIgdbApikey(apikey.Value)} />
					</label>
					<div className="col-sm-10">
						{apikey.Value !== '' ? (
							<SecretReveal secret={apikey.Value} />
						) : (
							<span>
								Not set - unable to fetch metadata.{' '}
								<a href="https://www.igdb.com/api" target="_blank">
									Register for an API key
								</a>
								&nbsp; (&amp; get more info).
							</span>
						)}
					</div>
				</div>
			</div>
		);
	}

	private fetchData() {
		this.state.apikeyTmdb.load(() => getConfig(CfgTheMovieDbApikey));
		this.state.apikeyIgdb.load(() => getConfig(CfgIgdbApikey));
	}
}
