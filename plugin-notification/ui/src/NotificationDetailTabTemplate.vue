<template>
  <div class="notification-detail-tab-template">
    <el-input v-model="internalTitle" class="title" placeholder="Title" @input="onTitleChange"/>
    <textarea :value="form.template" class="simple-mde" ref="simpleMDERef"/>
  </div>
</template>

<script lang="ts">
import {defineComponent, onMounted, onBeforeUnmount, ref, computed} from 'vue';

export default defineComponent({
  name: 'NotificationDetailTabTemplate',
  props: {
    form: {
      type: Object,
      default: () => {
        return {};
      },
    },
  },
  emits: [
    'template-change',
    'title-change',
  ],
  setup(props, {emit}) {
    const simpleMDERef = ref();

    const simpleMDE = ref();

    const internalTitle = ref();

    onMounted(() => {
      simpleMDE.value = new window.SimpleMDE({
        element: simpleMDERef.value,
        spellChecker: false,
        placeholder: 'Template content',
      });
      simpleMDE.value.codemirror.on('change', () => {
        emit('template-change', simpleMDE.value.value());
      });

      const {title} = props.form;
      internalTitle.value = title;

      const codeMirrorEl = document.querySelector('.CodeMirror');
      if (!codeMirrorEl) return;
      codeMirrorEl.setAttribute('style', 'height: 100%; min-height: 100%;');
    });

    onBeforeUnmount(() => {
      simpleMDE.value.toTextArea();
      simpleMDE.value = null;
    });

    const onTitleChange = (value) => {
      emit('title-change', value);
    };

    return {
      simpleMDERef,
      internalTitle,
      onTitleChange,
    };
  },
});
</script>

<style scoped>
.notification-detail-tab-template {
  /*min-height: 100%;*/
  display: flex;
  flex-direction: column;
}
.notification-detail-tab-template .title {
  /*margin-bottom: 20px;*/
}

.notification-detail-tab-template .title >>> .el-input__inner{
  border: none;
}

.notification-detail-tab-template >>> .editor-toolbar {
  border-radius: 0;
  border-left: none;
  border-right: none;
}

.notification-detail-tab-template >>> .CodeMirror {
  border-radius: 0;
}
</style>
