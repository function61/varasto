import { Panel, Well } from 'f61ui/component/bootstrap';
import { CommandIcon } from 'f61ui/component/CommandButton';
import { Loading } from 'f61ui/component/loading';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import {
	ConfigSetFuseServerBaseurl,
	ConfigSetNetworkShareBaseUrl,
} from 'generated/stoserver/stoservertypes_commands';
import { getConfig } from 'generated/stoserver/stoservertypes_endpoints';
import {
	CfgFuseServerBaseUrl,
	CfgNetworkShareBaseUrl,
	ConfigValue,
} from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface FuseServerPageState {
	baseUrl?: ConfigValue;
	networkShareBaseUrl?: ConfigValue;
}

export default class FuseServerPage extends React.Component<{}, FuseServerPageState> {
	state: FuseServerPageState = {};

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		return (
			<SettingsLayout title="FUSE server &amp; network folders" breadcrumbs={[]}>
				<Panel heading="Settings">{this.renderEditForm()}</Panel>
			</SettingsLayout>
		);
	}

	private renderEditForm() {
		const baseUrl = this.state.baseUrl;
		const networkShareBaseUrl = this.state.networkShareBaseUrl;

		if (!baseUrl || !networkShareBaseUrl) {
			return <Loading />;
		}

		return (
			<div className="form-horizontal">
				<div className="form-group">
					<label className="col-sm-2 control-label">
						FUSE server base URL
						<CommandIcon command={ConfigSetFuseServerBaseurl(baseUrl.Value)} />
					</label>
					<div className="col-sm-10">
						{baseUrl.Value !== ''
							? baseUrl.Value
							: 'Not set - unable to mount network folders'}
					</div>
				</div>

				<div className="form-group">
					<label className="col-sm-2 control-label">
						Network share base URL
						<CommandIcon
							command={ConfigSetNetworkShareBaseUrl(networkShareBaseUrl.Value)}
						/>
					</label>
					<div className="col-sm-10">
						{networkShareBaseUrl.Value !== '' ? networkShareBaseUrl.Value : 'Not set'}
					</div>
				</div>

				<Well>
					<p>FUSE is technology in Linux where we can easily define filesystems.</p>

					<p>
						Varasto supports projecting Varasto collections over FUSE as read-only
						files. Great use cases are direct-streaming videos. If you want transcoded
						videos, those you can view directly from Varasto's web ui.
					</p>

					<p>
						You need to run this Varasto-FUSE process separately from the main Varasto
						binary.
					</p>

					<p>
						You can then export the filesystem over Samba as a read-only network folder
						to other computers in the network.
					</p>
				</Well>
			</div>
		);
	}

	private async fetchData() {
		const [baseUrl, networkShareBaseUrl] = await Promise.all([
			getConfig(CfgFuseServerBaseUrl),
			getConfig(CfgNetworkShareBaseUrl),
		]);

		this.setState({ baseUrl, networkShareBaseUrl });
	}
}
