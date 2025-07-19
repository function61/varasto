import { Result } from 'f61ui/component/result';
import { WarningAlert } from 'f61ui/component/alerts';
import { CommandInlineForm } from 'f61ui/component/CommandButton';
import { CollapsePanel } from 'f61ui/component/bootstrap';
import { ConfigSetGrafanaURL } from 'generated/stoserver/stoservertypes_commands';
import { getConfig } from 'generated/stoserver/stoservertypes_endpoints';
import { CfgGrafanaUrl, ConfigValue } from 'generated/stoserver/stoservertypes_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';
import { serverInfoUrl } from 'generated/frontend_uiroutes';

interface MetricsPageState {
	grafanaUrl: Result<ConfigValue>;
}

export default class MetricsPage extends React.Component<{}, MetricsPageState> {
	state: MetricsPageState = {
		grafanaUrl: new Result<ConfigValue>((_) => {
			this.setState({ grafanaUrl: _ });
		}),
	};

	componentDidMount() {
		this.fetchData();
	}

	componentWillReceiveProps() {
		this.fetchData();
	}

	render() {
		const [grafanaUrl, loadingOrError] = this.state.grafanaUrl.unwrap();
		if (!grafanaUrl || loadingOrError) {
			return loadingOrError;
		}
		return (
			<AppDefaultLayout
				title="Metrics"
				breadcrumbs={[
					{
						url: serverInfoUrl(),
						title: 'Settings',
					},
				]}>
				<div className="row">
					<div className="col-md-12">
						{this.state.grafanaUrl.draw((_) => this.renderGrafanaEmbed(_.Value))}
						{this.state.grafanaUrl.draw((_) => this.renderConfig(_))}
					</div>
				</div>
			</AppDefaultLayout>
		);
	}

	private renderGrafanaEmbed(grafanaUrl: string) {
		if (!grafanaUrl) {
			return <WarningAlert>Grafana integration not configured.</WarningAlert>;
		}

		return <iframe src={grafanaUrl} style={{ width: '100%', height: '1000px', border: '0' }} />;
	}

	private renderConfig(grafanaUrl: ConfigValue) {
		return (
			<CollapsePanel
				heading="Grafana integration configuration"
				openInitially={!grafanaUrl.Value}>
				<CommandInlineForm command={ConfigSetGrafanaURL(grafanaUrl.Value)} />
			</CollapsePanel>
		);
	}

	private fetchData() {
		this.state.grafanaUrl.load(() => getConfig(CfgGrafanaUrl));
	}
}
