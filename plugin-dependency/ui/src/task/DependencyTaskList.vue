<template>
  <cl-table
      :columns="tableColumns"
      :data="tableData"
      :page="tablePagination.page"
      :page-size="tablePagination.size"
      :total="tableTotal"
      :visible-buttons="['']"
      @pagination-change="onPaginationChange"
  />
  <cl-dialog
      :title="t('task.logs')"
      :visible="dialogVisible.logs"
      width="1200px"
      @confirm="onLogsClose"
      @close="onLogsClose"
  >
    <LogsView :logs="logs"/>
  </cl-dialog>
</template>

<script lang="ts">
import {computed, defineComponent, h, onBeforeUnmount, onMounted, ref, watch} from 'vue';
import {ClNodeType, ClTag, ClTaskStatus, ClTime, useRequest} from 'crawlab-ui';
import {useStore} from 'vuex';
import TaskAction from './TaskAction.vue';
import LogsView from './LogsView.vue';

const pluginName = 'dependency';
const t = (path) => window['_tp'](pluginName, path);
const _t = window['_t'];

const endpoint = '/plugin-proxy/dependency/tasks';

const {
  getList: getList_,
} = useRequest();

export default defineComponent({
  name: 'DependencyTaskList',
  components: {
    LogsView,
    TaskAction,
  },
  props: {
    type: {
      type: String,
    },
  },
  setup(props, {emit}) {
    const store = useStore();

    const dialogVisible = ref({
      logs: false,
    });

    const logs = ref([]);

    const getLogs = async (id) => {
      const res = await getList_(`${endpoint}/${id}/logs`);
      const {data} = res;
      logs.value = data.map(d => d.content);
    };

    let logsHandle;

    const onLogsOpen = async (id) => {
      await getLogs(id);
      dialogVisible.value.logs = true;
      logsHandle = setInterval(() => getLogs(id), 5000);
    };

    const onLogsClose = () => {
      dialogVisible.value.logs = false;
      clearInterval(logsHandle);
    };

    const allNodeDict = computed(() => store.getters[`node/allDict`]);

    const tableColumns = computed(() => [
      {
        key: 'action',
        label: t('task.table.columns.action'),
        icon: ['fa', 'hammer'],
        width: '120',
        value: (row) => {
          return h(TaskAction, {action: row.action});
        },
      },
      {
        key: 'node',
        label: t('task.table.columns.node'),
        icon: ['fa', 'server'],
        width: '120',
        value: (row) => {
          const n = allNodeDict.value.get(row.node_id);
          if (!n) return;
          return h(ClNodeType, {
            isMaster: n.is_master,
            label: n.name,
          });
        },
      },
      {
        key: 'status',
        label: t('task.table.columns.status'),
        icon: ['fa', 'check-square'],
        width: '120',
        value: (row) => {
          return h(ClTaskStatus, {status: row.status, error: row.error});
        },
      },
      {
        key: 'dep_names',
        label: t('task.table.columns.dependencies'),
        icon: ['fa', 'puzzle-piece'],
        width: '380',
        value: (row) => {
          if (!row.dep_names) return [];
          return row.dep_names.map(depName => {
            return h(ClTag, {label: depName});
          });
        },
      },
      {
        key: 'update_ts',
        label: t('task.table.columns.time'),
        icon: ['fa', 'clock'],
        width: '150',
        value: (row) => {
          return h(ClTime, {time: row.update_ts});
        },
      },
      {
        key: 'actions',
        label: _t('components.table.columns.actions'),
        fixed: 'right',
        width: '80',
        buttons: (row) => {
          return [
            {
              type: 'primary',
              icon: ['fa', 'file-alt'],
              tooltip: t('task.logs'),
              onClick: async (row) => {
                await onLogsOpen(row._id);
              },
            },
          ];
        },
        disableTransfer: true,
      },
    ]);

    const tableData = ref([]);

    const tablePagination = ref({
      page: 1,
      size: 10,
    });

    const tableTotal = ref(0);

    const onPaginationChange = (pagination) => {
      tablePagination.value = {...pagination};
    };

    const getList = async () => {
      const res = await getList_(`${endpoint}`, {
        ...tablePagination.value,
        conditions: [{
          key: 'type',
          op: 'eq',
          value: props.type,
        }]
      });
      const {data, total} = res;
      tableData.value = data || [];
      tableTotal.value = total || 0;
    };

    watch(() => tablePagination.value.size, getList);
    watch(() => tablePagination.value.page, getList);

    let handle;

    onMounted(async () => {
      await getList();
      handle = setInterval(getList, 5000);
    });

    onBeforeUnmount(() => {
      clearInterval(handle);
      clearInterval(logsHandle);
    });

    return {
      dialogVisible,
      tableColumns,
      tableData,
      tablePagination,
      tableTotal,
      onPaginationChange,
      getList,
      logs,
      onLogsClose,
      t,
    };
  },
});
</script>

<style scoped>

</style>
