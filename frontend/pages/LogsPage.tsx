import { Panel } from 'f61ui/component/bootstrap';
import { Loading } from 'f61ui/component/loading';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import { getLogs } from 'generated/stoserver/stoservertypes_endpoints';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface LogsPageState {
	logs?: string[];
}

export default class LogsPage extends React.Component<{}, LogsPageState> {
	state: LogsPageState = {};

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		return (
			<SettingsLayout title="Logs" breadcrumbs={[]}>
				<Panel heading="Logs">{this.renderLogs()}</Panel>
			</SettingsLayout>
		);
	}

	private renderLogs() {
		const logs = this.state.logs;

		if (!logs) {
			return <Loading />;
		}

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th>Line</th>
					</tr>
				</thead>
				<tbody>
					{logs.map((line) => (
						<tr>
							<td>{line}</td>
						</tr>
					))}
				</tbody>
			</table>
		);
	}

	private async fetchData() {
		const logs = await getLogs();

		this.setState({ logs });
	}
}
