import { RefreshButton } from 'component/refreshbutton';
import { Result } from 'f61ui/component/result';
import { Panel } from 'f61ui/component/bootstrap';
import { getLogs } from 'generated/stoserver/stoservertypes_endpoints';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface LogsPageState {
	logs: Result<string[]>;
}

export default class LogsPage extends React.Component<{}, LogsPageState> {
	state: LogsPageState = {
		logs: new Result<string[]>((_) => {
			this.setState({ logs: _ });
		}),
	};

	componentDidMount() {
		this.fetchData();
	}

	componentWillReceiveProps() {
		this.fetchData();
	}

	render() {
		const [logs, loadingOrError] = this.state.logs.unwrap();

		return (
			<SettingsLayout title="Logs" breadcrumbs={[]}>
				<Panel heading="Logs">
					<RefreshButton
						refresh={() => {
							this.fetchData();
						}}
					/>

					<table className="table table-striped table-hover">
						<thead>
							<tr>
								<th>Line</th>
							</tr>
						</thead>
						<tbody>
							{(logs || []).map((line) => (
								<tr>
									<td>{line}</td>
								</tr>
							))}
						</tbody>
						<tfoot>
							<tr>
								<td colSpan={99}>{loadingOrError}</td>
							</tr>
						</tfoot>
					</table>

					<RefreshButton
						refresh={() => {
							this.fetchData();
						}}
					/>
				</Panel>
			</SettingsLayout>
		);
	}

	private fetchData() {
		this.state.logs.loadWhileKeepingOldResult(() => getLogs());
	}
}
