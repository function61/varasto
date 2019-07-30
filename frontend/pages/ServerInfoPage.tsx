import {
	changeSensitivity,
	getMaxSensitivityFromLocalStorage,
	Sensitivity,
	sensitivityLabel,
} from 'component/sensitivity';
import { Panel } from 'f61ui/component/bootstrap';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { CommandButton } from 'f61ui/component/CommandButton';
import { Loading } from 'f61ui/component/loading';
import { shouldAlwaysSucceed, unrecognizedValue } from 'f61ui/utils';
import { DatabaseBackup } from 'generated/stoserver/stoservertypes_commands';
import { getHealth, getServerInfo } from 'generated/stoserver/stoservertypes_endpoints';
import { Health, HealthStatus, ServerInfo } from 'generated/stoserver/stoservertypes_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';

interface ServerInfoPageState {
	serverInfo?: ServerInfo;
	health?: Health;
	currSens: Sensitivity;
}

export default class ServerInfoPage extends React.Component<{}, ServerInfoPageState> {
	state: ServerInfoPageState = { currSens: getMaxSensitivityFromLocalStorage() };

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		return (
			<AppDefaultLayout title="Server info" breadcrumbs={[]}>
				<Panel heading="Server info">{this.renderInfo()}</Panel>
				<Panel heading="Health">{this.renderHealth()}</Panel>
				<Panel heading="Sensitivity">{this.renderSensitivitySelector()}</Panel>
			</AppDefaultLayout>
		);
	}

	private renderInfo() {
		const serverInfo = this.state.serverInfo;

		if (!serverInfo) {
			return <Loading />;
		}

		interface Item {
			h: string;
			t: string;
		}

		const items: Item[] = [
			{ h: 'Varasto version', t: serverInfo.AppVersion },
			{ h: 'Varasto uptime', t: serverInfo.AppUptime },
			{ h: 'Database size', t: bytesToHumanReadable(serverInfo.DatabaseSize) },
			{ h: 'Go version', t: serverInfo.GoVersion },
			{ h: 'Server OS / arch', t: `${serverInfo.ServerOs} / ${serverInfo.ServerArch}` },
			{ h: 'Goroutines', t: serverInfo.Goroutines.toString() },
			{ h: 'Heap memory', t: bytesToHumanReadable(serverInfo.HeapBytes) },
		];

		return (
			<table className="table table-striped table-hover">
				<tbody>
					{items.map((item) => (
						<tr>
							<th>{item.h}</th>
							<td>{item.t}</td>
						</tr>
					))}
				</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>
							<CommandButton command={DatabaseBackup()} />
						</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	private renderSensitivitySelector() {
		const sensitivityRadioChange = (e: React.ChangeEvent<HTMLInputElement>) => {
			changeSensitivity(+e.target.value);
			this.setState({ currSens: getMaxSensitivityFromLocalStorage() });
		};

		const oneSensitivityRadio = (sens: Sensitivity) => (
			<div key={sens}>
				<label>
					<input
						type="radio"
						name="changeSensitivityRadio"
						onChange={sensitivityRadioChange}
						value={sens}
						checked={sens === this.state.currSens}
					/>{' '}
					{sensitivityLabel(sens)}
				</label>
			</div>
		);

		return (
			<div>
				{oneSensitivityRadio(Sensitivity.FamilyFriendly)}
				{oneSensitivityRadio(Sensitivity.Sensitive)}
				{oneSensitivityRadio(Sensitivity.MyEyesOnly)}
			</div>
		);
	}

	private renderHealth() {
		const health = this.state.health;

		if (!health) {
			return <Loading />;
		}

		const rows: JSX.Element[] = [];

		const pushHealthNodeAsRow = (node: Health, indentLevel: number) => {
			rows.push(
				<tr>
					<td>{healthStatusToIcon(node.Health)}</td>
					<td style={{ paddingLeft: indentLevel * 32 + 'px' }}>{node.Title}</td>
					<td>{node.Details}</td>
				</tr>,
			);

			node.Children.forEach((childHealth) => {
				pushHealthNodeAsRow(childHealth, indentLevel + 1);
			});
		};

		pushHealthNodeAsRow(health, 0);

		return (
			<table className="table table-striped table-hover">
				<thead>
					<tr>
						<th />
						<th>Title</th>
						<th>Details</th>
					</tr>
				</thead>
				<tbody>{rows}</tbody>
			</table>
		);
	}

	private async fetchData() {
		const [serverInfo, health] = await Promise.all([getServerInfo(), getHealth()]);

		this.setState({ serverInfo, health });
	}
}

function healthStatusToIcon(input: HealthStatus): JSX.Element {
	switch (input) {
		case HealthStatus.Fail:
			return (
				<span className="alert alert-danger">
					<span className="glyphicon glyphicon-fire" />
				</span>
			);
		case HealthStatus.Warn:
			return (
				<span className="alert alert-warning">
					<span className="glyphicon glyphicon-warning-sign" />
				</span>
			);
		case HealthStatus.Pass:
			return (
				<span className="alert alert-success">
					<span className="glyphicon glyphicon-ok" />
				</span>
			);
		default:
			throw unrecognizedValue(input);
	}
}
